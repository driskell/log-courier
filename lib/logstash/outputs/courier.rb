# encoding: utf-8

# Copyright 2014 Jason Woods.
#
# This file is a modification of code from Logstash Forwarder.
# Copyright 2012-2013 Jordan Sissel and contributors.
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

module LogStash
  module Outputs
    # Send events using the Log Courier protocol
    class LogCourier < LogStash::Outputs::Base
      config_name 'courier'
      milestone 1

      # The list of addresses Log Courier should send to
      config :hosts, :validate => :array, :required => true

      # The port to connect to
      config :port, :validate => :number, :required => true

      # CA certificate for validation of the server
      config :ssl_ca, :validate => :path, :required => true

      # Client SSL certificate to use
      config :ssl_certificate, :validate => :path

      # Client SSL key to use
      config :ssl_key, :validate => :path

      # SSL key passphrase to use
      config :ssl_key_passphrase, :validate => :password

      # Maximum number of events to spool before forcing a flush
      config :spool_size, :validate => :number, :default => 1024

      # Maximum time to wait for a full spool before forcing a flush
      config :idle_timeout, :validate => :number, :default => 5

      public

      def register
        require 'log-courier/client'

        @client = LogCourier::Client.new(
          addresses:          @hosts,
          port:               @port,
          ssl_ca:             @ssl_ca,
          ssl_certificate:    @ssl_certificate,
          ssl_key:            @ssl_key,
          ssl_key_passphrase: @ssl_key_passphrase,
          spool_size:         @spool_size,
          idle_timeout:       @idle_timeout
        )
      end

      public

      def receive(event)
        return unless output?(event)
        if event == LogStash::SHUTDOWN
          @client.shutdown
          finished
          return
        end
        @client.publish event.to_hash
      end
    end
  end
end
