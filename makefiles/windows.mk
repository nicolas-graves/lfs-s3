WINDOWS=${OUTPUT}/windows
MSI_PACKAGER=windows_msi.xml

## Windows Packing
${WINDOWS}/: | ${OUTPUT}/
	@mkdir $@

${WINDOWS}/${PROJECT}.exe: ${GO_FILES} | ${WINDOWS}/
	@GOOS=windows GOARCH=amd64 ${COMPILE_COMMAND} -o $@ .

${WINDOWS}/${PROJECT}.wxs: packaging/windows/windows_msi.xml | ${WINDOWS}/
	@cat $< | sed 's|$$VERSION|${TAG}|' > $@

${WINDOWS}/${PROJECT}.wixobj: ${WINDOWS}/${PROJECT}.wxs ${WINDOWS}/${PROJECT}.exe
	@echo Building wixobj
	@docker run --rm \
		-v $(shell pwd)/output/windows:/wix dactiv/wix candle \
		${PROJECT}.wxs

${WINDOWS}/${PROJECT}_${TAG}.msi: ${WINDOWS}/${PROJECT}.wixobj
	@echo Building msi
	@docker run --rm \
		-v $(shell pwd)/output/windows:/wix \
		dactiv/wix light \
		${<F} -sval -out ${@F}
