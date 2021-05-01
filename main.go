//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
//   Copyright 2018 Binx.io B.V.
package main

import (
	"bytes"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"flag"
	"fmt"
	"github.com/binxio/gcloudconfig"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

var verbose bool

const oauth2scope = "https://www.googleapis.com/auth/cloud-platform"

type Main struct {
	ctx         context.Context
	project     string
	verbose     bool
	credentials *google.Credentials
	client      *secretmanager.Client
	clientError error

	name    string
	command []string
}

type GoogleSecretRef struct {
	name         *string            // of the environment variable
	secretName   *string            // in the secret manager
	defaultValue *string            // if one is specified
	destination  *string            // to write the value to, otherwise ""
	fileMode     os.FileMode        // file permissions
	template     *template.Template // to use, defaults to '{{.}}'
}

func (m *Main) initialize() {
	var useDefaultCredentials bool
	flag.StringVar(&m.project, "project", "", "`id` of the project to query")
	flag.BoolVar(&m.verbose, "verbose", false, "get debug output")
	flag.BoolVar(&useDefaultCredentials, "use-default-credentials", false, "and ignore gcloud configuration")
	flag.StringVar(&m.name, "name", "", "of the secret")
	flag.Parse()

	m.ctx = context.Background()
	m.command = flag.Args()

	if useDefaultCredentials || !gcloudconfig.IsGCloudOnPath() {
		if m.verbose {
			log.Printf("INFO: using default application credentials\n")
		}
		m.credentials, m.clientError = google.FindDefaultCredentials(m.ctx, oauth2scope)
	} else {
		if m.verbose {
			log.Printf("INFO: using credentials from gcloud configuration\n")
		}
		m.credentials, m.clientError = gcloudconfig.GetCredentials("")
	}

	if m.clientError != nil {
		if m.verbose {
			log.Printf("failed to get credentials, %s", m.clientError)
		}
		return
	}

	if project, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT"); ok {
		m.credentials.ProjectID = project
	}

	if m.project != "" {
		m.credentials.ProjectID = m.project
	}

	if m.credentials.ProjectID == "" {
		m.clientError = fmt.Errorf("no project specified and no default project set")
		return
	}

	m.client, m.clientError = secretmanager.NewClient(m.ctx, option.WithCredentials(m.credentials))
}

func main() {
	m := Main{}

	m.initialize()

	if m.name != "" {
		if name, err := m.expandSecretName(m.Expand(m.name)); err == nil {
			if value, err := m.getSecret(name); err == nil {
				fmt.Printf("%s", value)
			} else {
				log.Fatalf("ERROR: failed to get secret, %s", err)
			}
		} else {
			log.Fatalf("ERROR: failed to get secret, %s", err)
		}

	} else {
		if len(m.command) < 1 {
			log.Fatalf("ERROR: expected --name or a command to run")
		}
		m.execProcess()
	}
}

func (m *Main) Getenv(name string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	} else {
		if m.verbose {
			log.Printf("WARN: environment variable %s is not set, returning empty string", name)
		}
	}
	return ""
}

func (m *Main) Expand(value string) string {
	return os.Expand(value, m.Getenv)
}

func (m *Main) expandSecretName(path string) (string, error) {
	var project string
	var name string
	var version string

	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if match, _ := regexp.MatchString("projects/[^/]+/secrets/[^/]+/.*", path); match {
		return path, nil
	}

	if m.credentials != nil {
		project = m.credentials.ProjectID
	}

	parts := strings.Split(path, "/")
	switch len(parts) {
	case 1:
		name = parts[0]
		version = "latest"
	case 2:
		if match, _ := regexp.MatchString("([0-9]+|latest)", parts[1]); match {
			name = parts[0]
			version = parts[1]
		} else {
			project = parts[0]
			name = parts[1]
			version = "latest"
		}
	case 3:
		project = parts[0]
		name = parts[1]
		version = parts[2]
	default:
		return path, fmt.Errorf("invalid secret name specification: %s", path)
	}
	result := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, name, version)
	if m.verbose {
		log.Printf("secret name = %s\n", result)
	}
	return result, nil
}

// converts the environment variables in `environ` into a list of Google secret references.
func (m *Main) environmentToGoogleSecretReferences(environ []string) ([]GoogleSecretRef, error) {
	result := make([]GoogleSecretRef, 0, 10)
	for i := 0; i < len(environ); i++ {
		name, value := toNameValue(environ[i])
		if strings.HasPrefix(value, "gcp:") {
			value := m.Expand(value)
			uri, err := url.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse environment variable %s, %s", name, err)
			}
			if uri.Host != "" {
				return nil, fmt.Errorf("environment variable %s has an gcp: uri, but specified a host. add a /.", name)
			}
			values, err := url.ParseQuery(uri.RawQuery)
			if err != nil {
				return nil, fmt.Errorf("environment variable %s has an invalid query syntax, %s", name, err)
			}

			defaultValue := values.Get("default")
			destination := values.Get("destination")
			var tpl *template.Template
			if values.Get("template") != "" {
				tpl, err = template.New("secret").Parse(values.Get("template"))
				if err != nil {
					return nil, fmt.Errorf("environment variable %s has an invalid template syntax, %s", name, err)
				}
			}
			var fileMode os.FileMode
			chmod := values.Get("chmod")
			if chmod != "" {
				if mode, err := strconv.ParseUint(chmod, 8, 32); err != nil {
					return nil, fmt.Errorf("chmod '%s' is not valid, %s", chmod, err)
				} else {
					fileMode = os.FileMode(mode)
				}
			}
			secretName, err := m.expandSecretName(uri.Path)
			if err != nil {
				log.Printf("ERROR: %s, %s", environ[i], err)
			}

			result = append(result, GoogleSecretRef{&name, &secretName,
				&defaultValue, &destination, os.FileMode(fileMode), tpl})
		}
	}
	return result, nil
}

// get the default value for the secret
func (m *Main) getDefaultValue(ref *GoogleSecretRef) (string, error) {
	if *ref.defaultValue != "" {
		if ref.template != nil {
			return m.formatValue(ref, ref.defaultValue), nil
		}
		return *ref.defaultValue, nil
	}

	if *ref.destination != "" {
		content, err := ioutil.ReadFile(*ref.destination)
		if err == nil {
			return string(content), nil
		}
		return "", fmt.Errorf("destination file does not exist to provide default value")
	}
	return "", fmt.Errorf("no default value available")
}

// retrieve all the secret values from refs and return the result as a name-value map.
func (m *Main) googleSecretReferencesToEnvironment(refs []GoogleSecretRef) (map[string]string, error) {
	result := make(map[string]string)
	for _, ref := range refs {
		result[*ref.name] = *ref.defaultValue
		value, err := m.getSecret(*ref.secretName)
		if err == nil {
			result[*ref.name] = m.formatValue(&ref, &value)
		} else {
			msg := fmt.Sprintf("failed to get secret %s, %s", *ref.name, err)
			if m.verbose {
				log.Printf("WARNING: %s", msg)
			}
			value, err := m.getDefaultValue(&ref)
			if err != nil {
				return nil, fmt.Errorf("ERROR: %s, %s\n", msg, err)
			}
			result[*ref.name] = value
		}

	}
	return result, nil
}

func (m *Main) formatValue(ref *GoogleSecretRef, value *string) string {
	var writer bytes.Buffer
	if ref.template == nil {
		return *value
	}

	if err := ref.template.Execute(&writer, value); err != nil {
		log.Fatalf("failed to format value of '%s' with template", *ref.name)
	}
	return writer.String()
}

// create a new environment from `env` with new values from `newEnv`
func (m *Main) updateEnvironment(env []string, newEnv map[string]string) []string {
	result := make([]string, 0, len(env))
	for i := 0; i < len(env); i++ {
		name, _ := toNameValue(env[i])
		if newValue, ok := newEnv[name]; ok {
			result = append(result, fmt.Sprintf("%s=%s", name, newValue))
		} else {
			result = append(result, env[i])
		}
	}
	return result
}

// write the value of each reference to the specified destination file
func (m *Main) writeParameterValues(refs []GoogleSecretRef, env map[string]string) error {
	for _, ref := range refs {
		if *ref.destination != "" {
			f, err := os.Create(*ref.destination)
			if err != nil {
				return fmt.Errorf("failed to open file %s to write to, %s", *ref.destination, err)
			}
			_, err = f.WriteString(env[*ref.name])
			if err != nil {
				return fmt.Errorf("failed to write to file %s, %s", *ref.destination, err)
			}
			err = f.Close()
			if err != nil {
				return fmt.Errorf("failed to close file %s, %s", *ref.destination, err)
			}

			if ref.fileMode != 0 {
				err := os.Chmod(*ref.destination, ref.fileMode)
				if err != nil {
					return fmt.Errorf("failed to chmod file %s to %s, %s", *ref.destination, ref.fileMode, err)
				}
			}
		}
	}
	return nil
}
func (m *Main) replaceDestinationReferencesWithURL(refs []GoogleSecretRef, env map[string]string) map[string]string {
	for _, ref := range refs {
		if *ref.destination != "" {
			env[*ref.name] = fmt.Sprintf("%s", *ref.destination)
		}
	}
	return env
}

// execute the `command` with the environment set to actual values from the parameter store
func (m *Main) execProcess() {

	refs, err := m.environmentToGoogleSecretReferences(os.Environ())
	if err != nil {
		log.Fatal(err)
	}
	newEnv, err := m.googleSecretReferencesToEnvironment(refs)
	if err != nil {
		log.Fatal(err)
	}

	err = m.writeParameterValues(refs, newEnv)
	if err != nil {
		log.Fatal(err)
	}

	newEnv = m.replaceDestinationReferencesWithURL(refs, newEnv)

	if len(m.command) == 1 && m.command[0] == "noop" {
		if m.verbose {
			log.Printf("INFO: noop")
		}
		return
	}

	program, err := exec.LookPath(m.command[0])
	if err != nil {
		log.Fatalf("could not find program %s on path, %s", m.command[0], err)
	}

	err = syscall.Exec(program, m.command, m.updateEnvironment(os.Environ(), newEnv))
	if err != nil {
		log.Fatal(err)
	}
}

func (m *Main) getSecret(name string) (string, error) {

	if m.clientError != nil {
		return "", m.clientError
	}
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := m.client.AccessSecretVersion(m.ctx, accessRequest)
	if err != nil {
		return "", err
	}

	return string(result.Payload.Data), nil
}

// get the name and variable of a environment entry in the form of <name>=<value>
func toNameValue(envEntry string) (string, string) {
	result := strings.SplitN(envEntry, "=", 2)
	return result[0], result[1]
}
