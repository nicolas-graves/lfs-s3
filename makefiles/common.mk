
PROJECT=lfs-s3
GO_FILE=$(shell find . -type f -name "*.go")
TAG=$(shell git describe --tags | sed 's|^v||' | sed 's|\(\.*\)-.*|\1|')
COMPILE_COMMAND=go build -ldflags="-X main.Version=${TAG}"

OUTPUT=output
PACKAGE_FILES=package_files

.variables:
	@echo PROJECT         : ${PROJECT}
	@echo GO_FILE         : ${GO_FILE}
	@echo TAG             : ${TAG}
	@echo COMPILE_COMMAND : ${COMPILE_COMMAND}
	@echo OUTPUT          : ${OUTPUT}
	@echo OSX             : ${OSX}
	@echo AMD_OSX         : ${AMD_OSX}
	@echo ARM_OSX         : ${ARM_OSX}
	@echo WINDOWS         : ${WINDOWS}

