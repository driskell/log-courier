# Copyright 2014-2021 Jason Woods.
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

require 'logstash/version'
require 'rubygems/version'

module LogStash
  module Outputs
    # Send events using the Log Courier protocol
    class Courier < LogStash::Outputs::Base
      config_name 'courier'

      # The list of addresses Log Courier should send to
      config :addresses, validate: :array, required: true

      # The port to connect to
      config :port, validate: :number, required: true

      # The transport type to use
      config :transport, validate: :string, default: 'tls'

      # CA certificate for validation of the server
      config :ssl_ca, validate: :path, required: true

      # Client SSL certificate to use
      config :ssl_certificate, validate: :path

      # Client SSL key to use
      config :ssl_key, validate: :path

      # SSL key passphrase to use
      config :ssl_key_passphrase, validate: :password

      # Set minimum TLS version
      config :min_tls_version, validate: :number, default: 1.2

      # Maximum number of events to spool before forcing a flush
      config :spool_size, validate: :number, default: 1024

      # Maximum time to wait for a full spool before forcing a flush
      config :idle_timeout, validate: :number, default: 5

      def register
        @logger.info 'Starting courier output'

        require 'log-courier/client'
        @client = LogCourier::Client.new(options)
        nil
      end

      def receive(event)
        return unless output?(event)

        @client.publish event.to_hash
        nil
      end

      def close
        @client.shutdown
        nil
      end

      private

      def options
        result = {}
        [
          :logger, :addresses, :port, :transport, :ssl_ca, :ssl_certificate, :ssl_key,
          :ssl_key_passphrase, :spool_size, :idle_timeout, :min_tls_version,
        ].each do |k|
          result[k] = send(k)
        end
        result
      end

      def addresses
        # TODO: Fix this naming inconsistency
        @hosts
      end
    end
  end
end
