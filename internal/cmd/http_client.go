package cmd

import "github.com/steipete/gogcli/internal/googleapi"

// outboundHTTPClient bounds response-header waits for unauthenticated requests.
var outboundHTTPClient = googleapi.NewBoundedHTTPClient()
