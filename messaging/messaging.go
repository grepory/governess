/* Messaging's responsibility is to provide convenient interfaces to facilitate
 * safe, structured message passing to avoid lost/dead messages. All
 * "wire-level" serialization/deserialization should occur within the messaging
 * package so that plugging different messaging subsystems in (or ripping them
 * out entirely) is easier.
 */
package messaging

import "github.com/op/go-logging"

var (
	logger = logging.MustGetLogger("messaging")
)
