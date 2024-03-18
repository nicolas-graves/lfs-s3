#!/bin/sh

SOURCE_DIR=$1
echo SOURCE_DIR $SOURCE_DIR

DMG_FILE=$2
echo DMG_FILE $DMG_FILE

DIR_SIZE=$(du -sm ${SOURCE_DIR} | sed 's/\(^\d\+\).*/\1/')
echo SIZE $DIR_SIZE

dd if=/dev/zero of=$DMG_FILE bs=1M count=$DIR_SIZE
mkfs.hfsplus -v Install $DMG_FILE
cp -av $SOURCE_DIR $MOUNT_FILE
