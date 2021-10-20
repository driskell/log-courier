# Copyright 2014-2021 Jason Woods and Contributors.
#
# This file is a modification of code from Ruby.
# Ruby is copyrighted free software by Yukihiro Matsumoto <matz@netlab.jp>.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# Name calculation from HELO/VERS
#
module LogCourier
  # Protocol
  module Protocol
    def self.parse_helo_vers(data)
      data = "\x00\x00\x00\x00\x00\x00\x00\x00" if data.length < 8

      flags, major_version, minor_version, patch_version, client = data.unpack('CCCCA4')
      client = case client
               when 'LCOR'
                 'Log Courier'
               when 'LCVR'
                 'Log Carver'
               when 'RYLC'
                 'Ruby Log Courier'
               else
                 'Unknown'
               end

      if major_version != 0 || minor_version != 0 || patch_version != 0
        version = "#{major_version}.#{minor_version}.#{patch_version}"
        client_version = "#{client} v#{version}"
      else
        version = ''
        client_version = client
      end

      {
        flags: flags,
        major_version: major_version,
        minor_version: minor_version,
        patch_version: patch_version,
        client: client,
        version: version,
        client_version: client_version,
      }
    end
  end
end
