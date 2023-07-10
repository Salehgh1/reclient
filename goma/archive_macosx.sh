#!/bin/bash
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script produces the following in the output location
# of the ninja rule:
# 1. lib/goma_inputprocessor.a: an archive with all .o files
#    relevant to goma input processing.
# 2. include/: All headers generated during the goma build.

set -vx

# OUTDIR is the output location of the goma build with ninja.
OUTDIR=$1

# INSTALLDIR is the destination directory where archives and
# generated headers should be written.
INSTALLDIR=$2

find $OUTDIR/obj/client -name "*.o" -print > $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/glog -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/lib -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/base -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/abseil/abseil -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/abseil/abseil_internal -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/chromium_base -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/protobuf/protobuf_full -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/jsoncpp -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/boringssl -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/zlib -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/zlib_x86_simd -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/zlib_adler32_simd -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/zlib_inflate_chunk_simd -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/zlib_crc32_simd -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/breakpad -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/libyaml -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
find $OUTDIR/obj/third_party/minizip -name "*.o" -print >> $INSTALLDIR/lib/lib.rsp
ar -rcs $INSTALLDIR/lib/goma_input_processor.a $(cat $INSTALLDIR/lib/lib.rsp)

# on mac, use_custom_libcxx = false,
# so no libc++abi and libc++.

cp -R $OUTDIR/gen/* $INSTALLDIR/include
