OSX=${OUTPUT}/osx
AMD_OSX=${OSX}/amd
ARM_OSX=${OSX}/arm



## OSX Packing
${OSX}/: | ${OUTPUT}/
	@mkdir $@

### AMD
${AMD_OSX}/: | ${OSX}/
	@mkdir $@

${AMD_OSX}/${PACKAGE_FILES}/: | ${AMD_OSX}/
	@mkdir $@

${AMD_OSX}/${PACKAGE_FILES}/${PROJECT}: ${GO_FILES} | ${AMD_OSX}/${PACKAGE_FILES}/
	@GOOS=darwin GOARCH=amd64 ${COMPILE_COMMAND} -o $@ .

### ARM (Apple Silicon)
${ARM_OSX}/: | ${OSX}/
	@mkdir $@

${ARM_OSX}/${PACKAGE_FILES}/: | ${ARM_OSX}/
	@mkdir $@

${ARM_OSX}/${PACKAGE_FILES}/${PROJECT}: ${GO_FILES} | ${ARM_OSX}/${PACKAGE_FILES}/
	@GOOS=darwin GOARCH=arm64 ${COMPILE_COMMAND} -o $@ .

## DMG package
${OSX}/%/${PROJECT}.dmg: ${OSX}/%/${PACKAGE_FILES}/${PROJECT} 
	@touch $@
	@dd if=/dev/zero of=$(shell pwd)/$@ bs=1M count=$(shell du -sm ${@D} | sed 's/\(^[0-9]\+\).*/\1/')
	@mkfs.hfsplus -v "Install lfs-s3" $@
	@mkdir -p ${@D}/mount
	@sudo mount $@ ${@D}/mount
	@sudo cp -av ${<D}/* ${@D}/mount
	@sudo umount ${@D}/mount
	@rmdir ${@D}/mount


