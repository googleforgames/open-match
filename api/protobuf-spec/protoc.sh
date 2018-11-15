python3 -m grpc_tools.protoc -I . --python_out=. --grpc_python_out=. mmlogic.proto
python3 -m grpc_tools.protoc -I . --python_out=. --grpc_python_out=. messages.proto
cp *pb2* $OM/examples/functions/python3/simple/.
