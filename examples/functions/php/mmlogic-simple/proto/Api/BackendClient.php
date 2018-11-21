<?php
// GENERATED CODE -- DO NOT EDIT!

namespace Api;

/**
 */
class BackendClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     * Run MMF once.  Return a matchobject that fits this profile.
     * INPUT: MatchObject message with these fields populated:
     *  - id
     *  - properties
     *  - [optional] roster, any fields you fill are available to your MMF.
     *  - [optional] pools, any fields you fill are available to your MMF.
     * OUTPUT: MatchObject message with these fields populated:
     *  - id
     *  - properties
     *  - error. Empty if no error was encountered
     *  - rosters, if you choose to fill them in your MMF. (Recommended)
     *  - pools, if you used the MMLogicAPI in your MMF. (Recommended, and provides stats)
     * @param \Messages\MatchObject $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateMatch(\Messages\MatchObject $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Backend/CreateMatch',
        $argument,
        ['\Messages\MatchObject', 'decode'],
        $metadata, $options);
    }

    /**
     * Continually run MMF and stream matchobjects that fit this profile until
     * client closes the connection.  Same inputs/outputs as CreateMatch.
     * @param \Messages\MatchObject $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function ListMatches(\Messages\MatchObject $argument,
      $metadata = [], $options = []) {
        return $this->_serverStreamRequest('/api.Backend/ListMatches',
        $argument,
        ['\Messages\MatchObject', 'decode'],
        $metadata, $options);
    }

    /**
     * Delete a matchobject from state storage manually. (Matchobjects in state
     * storage will also automatically expire after a while)
     * INPUT: MatchObject message with the 'id' field populated. 
     * (All other fields are ignored.)
     * @param \Messages\MatchObject $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteMatch(\Messages\MatchObject $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Backend/DeleteMatch',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

    /**
     * Call fors communication of connection info to players. 
     *
     * Write the connection info for the list of players in the
     * Assignments.messages.Rosters to state storage.  The FrontendAPI is
     * responsible for sending anything sent here to the game clients.
     * Sending a player to this function kicks off a process that removes
     * the player from future matchmaking functions by adding them to the 
     * 'deindexed' player list and then deleting their player ID from state storage
     * indexes.
     * INPUT: Assignments message with these fields populated:
     *  - connection_info, anything you write to this string is sent to Frontend API 
     *  - rosters. You can send any number of rosters, containing any number of
     *     player messages. All players from all rosters will be sent the connection_info.
     *     The only field in the Player object that is used by CreateAssignments is
     *     the id field.  All others are silently ignored.
     * @param \Messages\Assignments $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateAssignments(\Messages\Assignments $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Backend/CreateAssignments',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

    /**
     * Remove DGS connection info from state storage for players. 
     * INPUT: Roster message with the 'players' field populated. 
     *    The only field in the Player object that is used by
     *    DeleteAssignments is the 'id' field.  All others are silently ignored.  If
     *    you need to delete multiple rosters, make multiple calls.
     * @param \Messages\Roster $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteAssignments(\Messages\Roster $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.Backend/DeleteAssignments',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

}
