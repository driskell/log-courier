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
  module Inputs
    # Receive events over the Log Courier protocol
    class Courier < LogStash::Inputs::Base
      config_name 'courier'

      default :codec, 'plain'

      # The IP address to listen on
      config :host, validate: :string, default: '0.0.0.0'

      # The port to listen on
      config :port, validate: :number, required: true

      # The transport type to use
      config :transport, validate: :string, default: 'tls'

      # SSL certificate to use
      config :ssl_certificate, validate: :path

      # SSL key to use
      config :ssl_key, validate: :path

      # SSL key passphrase to use
      config :ssl_key_passphrase, validate: :password

      # Whether or not to verify client certificates
      config :ssl_verify, validate: :boolean, default: false

      # When verifying client certificates, also trust those signed by the
      # system's default CA bundle
      config :ssl_verify_default_ca, validate: :boolean, default: false

      # CA certificate to use when verifying client certificates
      config :ssl_verify_ca, validate: :path

      # Set minimum TLS version
      config :min_tls_version, validate: :number, default: 1.2

      # Max packet size
      config :max_packet_size, validate: :number

      # Add additional fields to events that identity the peer
      #
      # This setting is only effective with the tcp and tls transports
      #
      # "peer" identifies the source host and port
      # "peer_ssl_cn" contains the client certificate hostname for TLS peers
      # using client certificates
      config :add_peer_fields, validate: :boolean

      def register
        @logger.info(
          'Starting courier input listener',
          address: "#{@host}:#{@port}",
        )

        require 'log-courier/server'
        @log_courier = LogCourier::Server.new options
        nil
      end

      # Logstash < 2.0.0 shutdown raises LogStash::ShutdownSignal in this thread
      # The exception implicitly stops the log-courier gem using an ensure block
      # and is then caught by the pipeline worker - so we needn't do anything here
      def run(output_queue)
        @log_courier.run do |event|
          event['tags'] = [event['tags']] if event.key?('tags') && !event['tags'].is_a?(Array)
          event = LogStash::Event.new(event)
          decorate event
          output_queue << event
        end
        nil
      end

      def close
        @log_courier.stop
        nil
      end

      private

      def options
        result = {}
        add_plugin_options result
        add_override_options result
      end

      def add_plugin_options(result)
        [
          :logger, :address, :port, :transport, :ssl_certificate, :ssl_key,
          :ssl_key_passphrase, :ssl_verify, :ssl_verify_default_ca,
          :ssl_verify_ca, :min_tls_version,
        ].each do |k|
          result[k] = send(k)
        end
        result
      end

      def add_override_options(result)
        # Honour the defaults in the LogCourier gem
        [:max_packet_size, :add_peer_fields].each do |k|
          result[k] = send(k) unless send(k).nil?
        end
        result
      end

      def address
        # TODO: Fix this naming inconsistency
        @host
      end
    end
  end
end
