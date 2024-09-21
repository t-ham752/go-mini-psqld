package server

type tcpServerOption func(*TCPServer)

func WithServerVersion(version string) tcpServerOption {
	return func(s *TCPServer) {
		s.serverVersion = version
	}
}

func WithTimeZone(timezone string) tcpServerOption {
	return func(s *TCPServer) {
		s.timezone = timezone
	}
}

// implement more options
