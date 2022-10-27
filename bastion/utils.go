package bastion

import (
	"fmt"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/errortypes"

	"github.com/ozean12/pritunl-zero/utils"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

func DockerMatchContainer(a, b string) bool {
	if len(b) > len(a) {
		a, b = b, a
	}
	return strings.HasPrefix(a, b)
}

func DockerGetName(authrId primitive.ObjectID) string {
	return fmt.Sprintf("pritunl-bastion-%s", authrId.Hex())
}

func DockerGetRunning() (running map[string]primitive.ObjectID, err error) {
	running = map[string]primitive.ObjectID{}

	output, err := utils.ExecOutput("",
		"docker", "ps", "-a", "--format", "{{.Names}}:{{.ID}}")
	if err != nil {
		return
	}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(fields) != 2 {
			continue
		}

		name := fields[0]
		containerId := fields[1]

		if len(name) != 40 || !strings.HasPrefix(name, "pritunl-bastion-") {
			continue
		}

		authrId, e := primitive.ObjectIDFromHex(name[16:])
		if e != nil {
			err = &errortypes.ParseError{
				errors.Wrap(e, "bastion: Failed to parse ObjectID"),
			}
			return
		}

		running[containerId] = authrId
	}

	return
}

func DockerRemove(containerId string) (err error) {
	_, err = utils.ExecOutputLogged(nil, "docker", "rm", "-f", containerId)
	if err != nil {
		return
	}

	return
}
