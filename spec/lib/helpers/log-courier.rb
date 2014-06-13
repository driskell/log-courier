# Copyright 2014 Jason Woods.
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

# Common helpers for testing the courier
shared_context 'Helpers_Log_Courier' do
  before :each do
    @config = File.open(File.join(TEMP_PATH, 'config'), 'w')

    @log_courier = nil
  end

  after :each do
    shutdown

    File.unlink(@config.path) if File.file?(@config.path)

    # This is important or the state will not be clean for the following test
    File.unlink('.log-courier') if File.file?('.log-courier')
  end

  def startup(config: nil, args: '', mode: 'r')
    # A standard configuration when we don't want anything special
    if config.nil?
      config = <<-config
      {
        "network": {
          "ssl ca": "#{@ssl_cert.path}",
          "servers": [ "127.0.0.1:#{server_port}" ],
          "timeout": 15,
          "reconnect": 1
        },
        "files": [
          {
            "paths": [ "#{TEMP_PATH}/logs/log-*" ]
          }
        ]
      }
      config
    end

    puts "Starting with configuration:\n#{config}"

    if @config.closed?
      # Reopen the config and rewrite it
      @config.reopen(@config.path, 'w')
    end
    @config.puts config
    @config.close

    # Start LSF
    @log_courier = IO.popen("bin/log-courier -config #{@config.path}" + (args.empty? ? '' : ' ' + args), mode)

    # Needs some time to startup - to ensure when we create new files AFTER this, they are not detected during startup
    sleep STARTUP_WAIT_TIME
  end

  def shutdown
    return if @log_courier.nil?
    # When shutdown is implemented this becomes TERM/QUIT
    Process.kill('KILL', @log_courier.pid)
    Process.wait(@log_courier.pid)
    @log_courier = nil
  end
end
