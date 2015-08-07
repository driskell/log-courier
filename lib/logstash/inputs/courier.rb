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

require 'logstash/version'
require 'rubygems/version'

module LogStash
  module Inputs
    # Receive events over the Log Courier protocol
    class Courier < LogStash::Inputs::Base
      config_name 'courier'

      # Compatibility with Logstash 1.4 requires milestone
      if Gem::Version.new(LOGSTASH_VERSION) < Gem::Version.new('1.5.0')
        milestone 2
      end

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

      # Curve secret key
      config :curve_secret_key, validate: :string

      # Max packet size
      config :max_packet_size, validate: :number

      # The size of the internal queue for each peer
      #
      # Sent payloads will be dropped when the queue is full
      #
      # This setting should max the max_pending_payloads Log Courier
      # configuration
      config :peer_recv_queue, validate: :number

      # Add additional fields to events that identity the peer
      #
      # This setting is only effective with the tcp and tls transports
      #
      # "peer" identifies the source host and port
      # "peer_ssl_cn" contains the client certificate hostname for TLS peers
      # using client certificates
      config :add_peer_fields, validate: :boolean

      public

      def register
        @logger.info(
          'Starting courier input listener',
          address: "#{@host}:#{@port}"
        )

        require 'log-courier/server'
        @log_courier = LogCourier::Server.new options
      end

      def run(output_queue)
        @log_courier.run do |event|
          if event.key?('tags') && !event['tags'].is_a?(Array)
            event['tags'] = [event['tags']]
          end
          event = LogStash::Event.new(event)
          decorate event
          output_queue << event
        end
      end

      private

      def options
        result = {}

        [
          :logger, :address, :port, :transport, :ssl_certificate, :ssl_key,
          :ssl_key_passphrase, :ssl_verify, :ssl_verify_default_ca,
          :ssl_verify_ca, :curve_secret_key
        ].each do |k|
          result[k] = send(k)
        end

        add_override_options result
      end

      def add_override_options(result)
        # Honour the defaults in the LogCourier gem
        [:max_packet_size, :peer_recv_queue, :add_peer_fields].each do |k|
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
