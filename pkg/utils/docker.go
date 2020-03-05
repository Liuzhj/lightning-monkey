package utils

import (
	"fmt"
	"strings"
)

func GetDockerRepositoryName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '/'
	})
	if len(parts) == 1 {
		return "docker.io"
	}
	//repo + image name
	if len(parts) == 2 {
		return fmt.Sprintf("docker.io/%s", parts[0])
	}
	return name[:strings.LastIndex(name, "/")]
}
