#!/usr/bin/env python3
"""
Maestro gRPC Adapter for FastAPI
Wraps existing FastAPI endpoints to be callable via gRPC
"""

import grpc
import json
import requests
from concurrent import futures
from typing import Any, Dict
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class FastAPIAdapter:
    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url
        
    def Execute(self, request, context):
        """Execute a method on the FastAPI service"""
        try:
            # Parse the request
            method = request.method
            payload = json.loads(request.payload) if request.payload else {}
            
            logger.info(f"Executing {method} with payload: {payload}")
            
            # Map gRPC method to FastAPI endpoint
            endpoint_map = {
                "CreateUser": "/api/users",
                "DeleteUser": "/api/users/{user_id}",
                "GetUser": "/api/users/{user_id}",
                "UpdateUser": "/api/users/{user_id}",
            }
            
            endpoint = endpoint_map.get(method, f"/api/{method.lower()}")
            
            # Replace path parameters
            for key, value in payload.items():
                endpoint = endpoint.replace(f"{{{key}}}", str(value))
            
            # Determine HTTP method
            if method.startswith("Create"):
                response = requests.post(f"{self.base_url}{endpoint}", json=payload)
            elif method.startswith("Delete"):
                response = requests.delete(f"{self.base_url}{endpoint}")
            elif method.startswith("Update"):
                response = requests.put(f"{self.base_url}{endpoint}", json=payload)
            else:
                response = requests.get(f"{self.base_url}{endpoint}", params=payload)
            
            # Return response
            return {
                "success": response.status_code < 400,
                "data": response.json() if response.text else {},
                "error": str(response.text) if response.status_code >= 400 else ""
            }
            
        except Exception as e:
            logger.error(f"Error executing {request.method}: {e}")
            return {
                "success": False,
                "data": {},
                "error": str(e)
            }
    
    def Compensate(self, request, context):
        """Execute compensation logic"""
        return self.Execute(request, context)
    
    def HealthCheck(self, request, context):
        """Check service health"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            return {
                "healthy": response.status_code == 200,
                "message": "Service is healthy"
            }
        except:
            return {
                "healthy": False,
                "message": "Service is not responding"
            }

if __name__ == "__main__":
    # Start gRPC server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    adapter = FastAPIAdapter()
    
    # Register the service (you'll need to generate Python proto files)
    # maestro_pb2_grpc.add_MaestroServiceServicer_to_server(adapter, server)
    
    server.add_insecure_port('[::]:50051')
    server.start()
    logger.info("FastAPI Adapter listening on port 50051")
    server.wait_for_termination()