package publisher

// EndpointRemote is a structure used by transports to communicate back with
// the Publisher. It also contains the associated address information.
type EndpointRemote struct {
  sink     *EndpointSink
  endpoint *Endpoint

  // The associated address pool
  AddressPool *AddressPool
}

// Ready is called by a transport to signal it is ready for events.
// This should be triggered once connection is successful and the transport is
// ready to send data. It should NOT be called again until the transport
// receives data, otherwise the call may block.
func (e *EndpointRemote) Ready() {
	e.sink.ReadyChan <- e.endpoint
}

// ResponseChan returns the channel that responses should be sent on
func (e *EndpointRemote) ResponseChan() chan<- *EndpointResponse {
	return e.sink.ResponseChan
}

// NewResponse creates a response wrapper linked to the endpoint that can be
// sent to the Publisher over the response channel
func (e *EndpointRemote) NewResponse(response interface{}) *EndpointResponse {
  return &EndpointResponse{e.endpoint, response}
}

// Fail is called by a transport to signal an error has occurred, and that all
// pending payloads should be returned to the publisher for retransmission
// elsewhere.
func (e *EndpointRemote) Fail(err error) {
	e.sink.FailChan <- &EndpointFailure{e.endpoint, err}
}
