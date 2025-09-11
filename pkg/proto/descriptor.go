package proto

import "google.golang.org/grpc"

var MaestroServiceDesc = grpc.ServiceDesc{
	ServiceName: "maestro.v1.MaestroService",
	HandlerType: (*MaestroServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Execute",
			Handler:    executeHandler,
		},
		{
			MethodName: "Compensate",
			Handler:    compensateHandler,
		},
		{
			MethodName: "HealthCheck",
			Handler:    healthCheckHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "maestro.proto",
}
