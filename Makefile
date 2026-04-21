PROTO_DIR = proto

PROTO_FILES_LECTURER = proto/lecturer/lecturer.proto
OUT_LECTURER = .

PROTO_FILES_LECTURE = proto/lecture/lecture.proto
OUT_LECTURE = .

PROTO_FILES_LOCATION = proto/location/location.proto
OUT_LOCATION = .

PROTO_FILES_EVENT = proto/event/event.proto
OUT_EVENT = .

.PHONY: proto

proto: event lecturer 

event:
	protoc \
		--go_out=$(OUT_EVENT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_EVENT) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES_EVENT)

lecture:
	protoc \
		--go_out=$(OUT_LECTURE) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_LECTURE) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES_LECTURE)

lecturer:
	protoc \
		--go_out=$(OUT_LECTURER) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_LECTURER) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES_LECTURER)

location:
	protoc \
		--go_out=$(OUT_LOCATION) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_LOCATION) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES_LOCATION)

