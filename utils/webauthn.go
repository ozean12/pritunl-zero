package utils

import (
	"fmt"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/pritunl/webauthn/protocol"
)

func ParseWebauthnError(err error) (newErr error) {
	if e, ok := err.(*protocol.Error); ok {
		newErr = &errortypes.AuthenticationError{
			errors.Wrapf(
				err, "secondary: Webauthn error %s - %s - %s",
				e.Type, e.DevInfo, e.Details,
			),
		}
	} else {
		newErr = &errortypes.AuthenticationError{
			errors.Wrap(err, fmt.Sprintf(
				"secondary: Webauthn unknown error")),
		}
	}

	return
}
