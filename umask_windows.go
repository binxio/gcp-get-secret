//build +windows

package main

import "log"

func setUmask(mask int) int {
	log.Printf("INFO: umask %04d is not supported on Windows\n", mask)
	return 0
}
