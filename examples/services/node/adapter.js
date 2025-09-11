#!/usr/bin/env node
/**
 * Maestro gRPC Adapter for Next.js API Routes
 * Wraps Next.js API routes to be callable via gRPC
 */

const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const axios = require('axios');

class NextJSAdapter {
  constructor(baseUrl = 'http://localhost:3000') {
    this.baseUrl = baseUrl;
  }

  async execute(call, callback) {
    try {
      const { method, payload, correlation_id } = call.request;
      const data = JSON.parse(payload || '{}');
      
      console.log(`Executing ${method} with correlation_id: ${correlation_id}`);
      
      // Map gRPC method to Next.js API route
      const routeMap = {
        'CreateCustomer': '/api/customers',
        'DeleteCustomer': '/api/customers/[id]',
        'SendTemplate': '/api/emails/send',
        'SendOrderConfirmation': '/api/notifications/order',
      };
      
      let endpoint = routeMap[method] || `/api/${method.toLowerCase()}`;
      
      // Replace route parameters
      Object.keys(data).forEach(key => {
        endpoint = endpoint.replace(`[${key}]`, data[key]);
      });
      
      // Determine HTTP method and make request
      let response;
      if (method.startsWith('Create') || method.startsWith('Send')) {
        response = await axios.post(`${this.baseUrl}${endpoint}`, data);
      } else if (method.startsWith('Delete')) {
        response = await axios.delete(`${this.baseUrl}${endpoint}`);
      } else if (method.startsWith('Update')) {
        response = await axios.put(`${this.baseUrl}${endpoint}`, data);
      } else {
        response = await axios.get(`${this.baseUrl}${endpoint}`, { params: data });
      }
      
      callback(null, {
        success: true,
        data: JSON.stringify(response.data),
        error: ''
      });
      
    } catch (error) {
      console.error(`Error executing ${call.request.method}:`, error.message);
      callback(null, {
        success: false,
        data: '{}',
        error: error.message
      });
    }
  }
  
  async compensate(call, callback) {
    // Reuse execute for compensation
    return this.execute(call, callback);
  }
  
  async healthCheck(call, callback) {
    try {
      const response = await axios.get(`${this.baseUrl}/api/health`, { timeout: 5000 });
      callback(null, {
        healthy: response.status === 200,
        message: 'Service is healthy'
      });
    } catch (error) {
      callback(null, {
        healthy: false,
        message: 'Service is not responding'
      });
    }
  }
}

// Start gRPC server
if (require.main === module) {
  const PROTO_PATH = __dirname + '/../../../proto/maestro.proto';
  
  const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
  });
  
  const maestroProto = grpc.loadPackageDefinition(packageDefinition).maestro.v1;
  const server = new grpc.Server();
  const adapter = new NextJSAdapter();
  
  server.addService(maestroProto.MaestroService.service, {
    Execute: adapter.execute.bind(adapter),
    Compensate: adapter.compensate.bind(adapter),
    HealthCheck: adapter.healthCheck.bind(adapter)
  });
  
  const PORT = process.env.GRPC_PORT || '50052';
  server.bindAsync(
    `0.0.0.0:${PORT}`,
    grpc.ServerCredentials.createInsecure(),
    () => {
      console.log(`Next.js Adapter listening on port ${PORT}`);
      server.start();
    }
  );
}

module.exports = NextJSAdapter;