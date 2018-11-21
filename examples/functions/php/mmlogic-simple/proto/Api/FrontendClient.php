<?php
// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// TODO: In a future version, these messages will be moved/merged with those in om_messages.proto
namespace Api;

/**
 */
class FrontendClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     * @param \Api\Group $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateRequest(\Api\Group $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Frontend/CreateRequest',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Api\Group $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteRequest(\Api\Group $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Frontend/DeleteRequest',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Api\PlayerId $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function GetAssignment(\Api\PlayerId $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Frontend/GetAssignment',
        $argument,
        ['\Messages\ConnectionInfo', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Api\PlayerId $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteAssignment(\Api\PlayerId $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Frontend/DeleteAssignment',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

}
