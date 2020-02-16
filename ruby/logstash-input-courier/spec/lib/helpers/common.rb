# encoding: utf-8

# Copyright 2014-2016 Jason Woods.
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

require 'thread'
require 'cabin'
require 'log-courier/client'
require 'log-courier/server'

# Common helpers for testing both ruby client and the courier
shared_context 'Helpers' do
  before :all do
    Thread.abort_on_exception = true

    @transport = 'tls'

    @ssl_cert = File.open(File.join(TEMP_PATH, 'ssl_cert'), 'w')
    @ssl_key = File.open(File.join(TEMP_PATH, 'ssl_key'), 'w')
    @ssl_csr = File.open(File.join(TEMP_PATH, 'ssl_csr'), 'w')

    # Generate the ssl key
    system('openssl req -config spec/lib/openssl.cnf -new -batch -keyout ' \
      "#{@ssl_key.path} -out #{@ssl_csr.path}")
    system('openssl x509 -extfile spec/lib/openssl.cnf -extensions ' \
      "extensions_section -req -days 365 -in #{@ssl_csr.path} -signkey " \
      "#{@ssl_key.path} -out #{@ssl_cert.path}")
  end

  after :all do
    [@ssl_cert, @ssl_key, @ssl_csr].each do |f|
      File.unlink(f.path) if File.file?(f.path)
    end
  end

  before :each do
    # When we add a file we log it here, so after we can remove them
    @files = []

    @event_queue = SizedQueue.new 10_000
    @servers = {}
    @server_counts = {}
    @server_threads = {}

    @clients = {}
  end

  after :each do
    shutdown_server
    shutdown_client
  end

  def create_logger(id, type)
    logger = Cabin::Channel.new
    logger.subscribe STDOUT
    logger['type'] = type.to_s
    logger['instance'] = id
    logger.level = :debug
    logger
  end

  def start_server(args = {})
    args = { id: '__default__', transport: nil }.merge!(args)
    id = args[:id]

    # Reset server for each test
    @servers[id] = LogCourier::Server.new(
      transport: args[:transport].nil? ? @transport : args[:transport],
      ssl_certificate: @ssl_cert.path, ssl_key: @ssl_key.path,
      logger: create_logger(id, :server)
    )

    @server_counts[id] = 0
    @server_threads[id] = start_server_thread(id)
    @servers[id]
  end

  def start_server_thread(id)
    Thread.new do
      begin
        @servers[id].run do |event|
          @server_counts[id] += 1
          @event_queue << event
        end
      rescue LogCourier::ShutdownSignal
        0
      end
    end
  end

  # A helper to shutdown a Log Courier server
  def shutdown_server(which = nil)
    which = if which.nil?
              @servers.keys
            else
              [which]
            end
    shutdown_multiple_servers(which)
  end

  def shutdown_multiple_servers(which)
    which.each do |id|
      @server_threads[id].raise LogCourier::ShutdownSignal
      @server_threads[id].join
      @server_threads.delete id
      @server_counts.delete id
      @servers.delete id
    end
  end

  def server_port(id = '__default__')
    @servers[id].port
  end

  def server_count(id = '__default__')
    @server_counts[id]
  end

  def start_client(address, port, args = {})
    args = {
      id:        '__default__',
      transport: nil
    }.merge!(args)

    id = args[:id]

    @clients[id] = LogCourier::Client.new(
      transport: args[:transport].nil? ? @transport : args[:transport],
      addresses: [address], port: port, ssl_ca: @ssl_cert.path,
      logger: create_logger(id, :client)
    )
  end

  def shutdown_client(which = nil)
    which = if which.nil?
              @clients.keys
            else
              [which]
            end
    shutdown_multiple_clients(which)
  end

  def shutdown_multiple_clients(which)
    which.each do |id|
      @clients[id].shutdown
      @clients.delete id
    end
  end

  def receive_and_check(total, &block)
    orig_total = total
    total = receive_and_check_events(total, &block)

    # Fancy calculation to give a nice "expected" output of expected # of events
    expect(orig_total - total).to eq orig_total
  end

  def receive_and_check_events(total, &block)
    waited = 0
    while total > 0 && waited <= EVENT_WAIT_COUNT
      if @event_queue.empty?
        sleep(EVENT_WAIT_TIME)
        waited += 1
        next
      end

      waited = 0
      total -= read_events(&block)
    end
    total
  end

  def read_events(&block)
    processed = 0
    until @event_queue.empty?
      e = @event_queue.pop
      processed += 1
      yield e unless block.nil?
    end
    processed
  end
end
