PROTO_DIR = proto

PROTO_FILES_PREDAVAC = proto/predavac-service/predavac-service.proto
OUT_PREDAVAC = proto/predavac-service

PROTO_FILES_DOGADJAJ = proto/dogadjaj-service/dogadjaj-service.proto
OUT_DOGADJAJ = proto/dogadjaj-service

.PHONY proto

proto: dogadjaj predavac 

dogadjaj:
	protoc \
		--proto-path=$(PROTO_DIR)
		--go-out=$(OUT_DOGADJAJ) \
		--go_opt=paths=source_relative \
		--go-grpc-out=$(OUT_DOGADJAJ)
		--go_grpc_opt=paths=source_relative \
		$(PROTO_FILES_DOGADJAJ)

predavac:
	protoc \
		--proto-path=$(PROTO_DIR) \
		--go-out=$(OUT_PREDAVAC) \
		--go_opt=paths=source_relative \
		--go_grpc_out=$(OUT_PREDAVAC)
		--go_grpc_opt=paths=source_relative \
		$(PROTO_FILES_PREDAVAC)

