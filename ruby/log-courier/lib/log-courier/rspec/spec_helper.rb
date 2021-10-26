# Copyright 2014-2021 Jason Woods.
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

require 'cabin'
require 'fileutils'
require 'log-courier/client'
require 'log-courier/server'

TEMP_PATH = File.join(File.dirname(__FILE__), 'tmp')
EVENT_WAIT_COUNT = 50
EVENT_WAIT_TIME = 0.5

# Common helpers for testing both ruby client and the courier
shared_context 'LogCourier' do
  before :all do
    Thread.abort_on_exception = true

    FileUtils.rm_r(TEMP_PATH) if File.directory?(TEMP_PATH)
    Dir.mkdir(TEMP_PATH)

    @ssl_cert = File.open(File.join(TEMP_PATH, 'ssl_cert'), 'w')
    @ssl_key = File.open(File.join(TEMP_PATH, 'ssl_key'), 'w')
    @ssl_csr = File.open(File.join(TEMP_PATH, 'ssl_csr'), 'w')

    # Generate the ssl key
    cnf_path = "#{File.dirname(__FILE__)}/openssl.cnf"
    system("openssl req -config #{cnf_path} -new -batch -keyout #{@ssl_key.path} -out #{@ssl_csr.path}")
    system(
      "openssl x509 -extfile #{cnf_path} -extensions extensions_section -req -days 365 -in #{@ssl_csr.path}" \
      " -signkey #{@ssl_key.path} -out #{@ssl_cert.path}",
    )
  end

  after :all do
    FileUtils.rm_r(TEMP_PATH) if File.directory?(TEMP_PATH)
  end

  before :each do
    @event_queue = SizedQueue.new 10_000

    @clients = {}
    @servers = {}
    @server_counts = {}
    @server_threads = {}
  end

  after :each do
    unless @servers.length.zero?
      id, = @servers.first
      raise "Server was not shutdown: #{id}"
    end
    unless @clients.length.zero?
      id, = @clients.first
      raise "Client was not shutdown: #{id}"
    end
  end

  def start_client(**args)
    args = {
      id: '__default__',
      transport: 'tls',
      addresses: ['127.0.0.1'],
    }.merge!(**args)

    args[:ssl_ca] = @ssl_cert.path if args[:transport] == 'tls'

    id = args[:id]
    args[:port] = server_port(id) unless args.key?(:port)

    logger = Cabin::Channel.new
    logger.subscribe $stdout
    logger['instance'] = "Client #{id}"
    logger.level = :debug

    # Reset server for each test
    @clients[id] = LogCourier::Client.new(
      logger: logger,
      **args,
    )
  end

  def shutdown_client(which = nil)
    which = if which.nil?
              @clients.keys
            else
              [which]
            end
    which.each do |id|
      @clients[id].shutdown
      @clients.delete id
    end
    nil
  end

  def start_server(**args)
    args = {
      id: '__default__',
      transport: 'tls',
    }.merge!(**args)

    if args[:transport] == 'tls'
      args[:ssl_certificate] = @ssl_cert.path
      args[:ssl_key] = @ssl_key.path
    end

    id = args[:id]

    logger = Cabin::Channel.new
    logger.subscribe $stdout
    logger['instance'] = "Server #{id}"
    logger.level = :debug

    raise 'Server already initialised' if @servers.key?(id)

    # Reset server for each test
    @servers[id] = LogCourier::Server.new(
      logger: logger,
      **args,
    )

    @server_counts[id] = 0
    @server_threads[id] = Thread.new do
      @servers[id].run do |event|
        @server_counts[id] += 1
        @event_queue << event
      end
    rescue LogCourier::ShutdownSignal
      0
    end
    @servers[id]
  end

  # A helper to shutdown a Log Courier server
  def shutdown_server(which = nil)
    which = if which.nil?
              @servers.keys
            else
              [which]
            end
    which.each do |id|
      @server_threads[id].raise LogCourier::ShutdownSignal
      @server_threads[id].join
      @server_threads.delete id
      @server_counts.delete id
      @servers.delete id
    end
    nil
  end

  # A helper to get the port a server is bound to
  def server_port(id = '__default__')
    @servers[id].port
  end

  # A helper to get number of events received on the server
  def server_count(id = '__default__')
    @server_counts[id]
  end

  def receive_and_check(args = {})
    args = {
      total: nil,
      check: true,
      check_file: true,
      check_order: true,
      host: nil,
    }.merge!(args)

    # Quick check of the total events we are expecting - but allow time to receive them
    total = if args[:total].nil?
              @files.reduce(0) do |sum, f|
                sum + f.count
              end
            else
              args[:total]
            end

    args.delete_if do |_, v|
      v.nil?
    end

    orig_total = total
    check = args[:check]

    waited = 0
    while total.positive? && waited <= EVENT_WAIT_COUNT
      if @event_queue.length.zero?
        sleep(EVENT_WAIT_TIME)
        waited += 1
        next
      end

      waited = 0
      until @event_queue.length.zero?
        e = @event_queue.pop
        total -= 1
        next unless check

        yield e
      end
    end

    # Fancy calculation to give a nice "expected" output of expected num of events
    expect(orig_total - total).to eq orig_total
    nil
  end
end
