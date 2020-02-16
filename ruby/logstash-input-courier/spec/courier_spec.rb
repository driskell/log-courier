# encoding: utf-8
require 'logstash/devutils/rspec/spec_helper'
require 'logstash/inputs/courier'
require 'lib/rspec_configure'

describe LogStash::Inputs::Courier do
  include_context 'Helpers'

  context 'logstash-input-courier' do
    it 'receives connections and generates events' do
      @plugin = LogStash::Inputs::Courier.new(
        'port' => 12_345,
        'ssl_certificate' => @ssl_cert.path,
        'ssl_key' => @ssl_key.path
      )
      @plugin.register
      @thread = Thread.new do
        @plugin.run @event_queue
      end

      client = start_client('127.0.0.1', 12_345)
      client.publish message: 'This is a test message'
      receive_and_check 1 do |e|
        expect(e.get('message')).to eq 'This is a test message'
      end

      shutdown_client
      @plugin.do_stop
      @thread.join
    end
  end
end
