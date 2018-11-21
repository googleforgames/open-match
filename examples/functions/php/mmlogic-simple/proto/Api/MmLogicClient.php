<?php
// GENERATED CODE -- DO NOT EDIT!

namespace Api;

/**
 * The MMLogic API provides utility functions for common MMF functionality, such
 * as retreiving profiles and players from state storage, writing results to state storage,
 * and exposing metrics and statistics.
 */
class MmLogicClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     *  Send GetProfile a match object with the ID field populated, it will return a
     *  'filled' one.
     *  Note: filters are assumed to have been checked for validity by the
     *  backendapi  when accepting a profile
     * @param \Messages\MatchObject $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function GetProfile(\Messages\MatchObject $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.MmLogic/GetProfile',
        $argument,
        ['\Messages\MatchObject', 'decode'],
        $metadata, $options);
    }

    /**
     * CreateProposal is called by MMFs that wish to write their results to
     * a proposed MatchObject, that can be sent out the Backend API once it has
     * been approved (by default, by the evaluator process).
     *  - adds all players in all Rosters to the proposed player ignore list
     *  - writes the proposed match to the provided key
     *  - adds that key to the list of proposals to be considered
     * INPUT: 
     *  * TO RETURN A MATCHOBJECT AFTER A SUCCESSFUL MMF RUN
     *    To create a match MatchObject message with these fields populated:
     *      - id, set to the value of the MMF_PROPOSAL_ID env var
     *      - properties
     *      - error.  You must explicitly set this to an empty string if your MMF
     *      - roster, with the playerIDs filled in the 'players' repeated field. 
     *      - [optional] pools, set to the output from the 'GetPlayerPools' call,
     *        will populate the pools with stats about how many players the filters
     *        matched and how long the filters took to run, which will be sent out
     *        the backend api along with your match results.
     *        was successful.
     *  * TO RETURN AN ERROR 
     *    To report a failure or error, send a MatchObject message with these
     *    these fields populated:
     *      - id, set to the value of the MMF_ERROR_ID env var. 
     *      - error, set to a string value describing the error your MMF encountered.
     *      - [optional] properties, anything you put here is returned to the
     *        backend along with your error.
     *      - [optional] rosters, anything you put here is returned to the
     *        backend along with your error.
     *      - [optional] pools, set to the output from the 'GetPlayerPools' call,
     *        will populate the pools with stats about how many players the filters
     *        matched and how long the filters took to run, which will be sent out
     *        the backend api along with your match results.
     * OUTPUT: a Result message with a boolean success value and an error string
     * if an error was encountered
     * @param \Messages\MatchObject $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateProposal(\Messages\MatchObject $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.MmLogic/CreateProposal',
        $argument,
        ['\Messages\Result', 'decode'],
        $metadata, $options);
    }

    /**
     * Player listing and filtering functions
     *
     * RetrievePlayerPool gets the list of players that match every Filter in the
     * PlayerPool, and then removes all players it finds in the ignore list.  It
     * combines the results, and returns the resulting player pool.
     * @param \Messages\PlayerPool $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function GetPlayerPool(\Messages\PlayerPool $argument,
      $metadata = [], $options = []) {
        return $this->_serverStreamRequest('/api.MmLogic/GetPlayerPool',
        $argument,
        ['\Messages\PlayerPool', 'decode'],
        $metadata, $options);
    }

    /**
     * Ignore List functions
     *
     * IlInput is an empty message reserved for future use.
     * @param \Messages\IlInput $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function GetAllIgnoredPlayers(\Messages\IlInput $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.MmLogic/GetAllIgnoredPlayers',
        $argument,
        ['\Messages\Roster', 'decode'],
        $metadata, $options);
    }

    /**
     * RetrieveIgnoreList retrieves players from the ignore list specified in the
     * config file under 'ignoreLists.proposedPlayers.key'.
     * @param \Messages\IlInput $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function ListIgnoredPlayers(\Messages\IlInput $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/api.MmLogic/ListIgnoredPlayers',
        $argument,
        ['\Messages\Roster', 'decode'],
        $metadata, $options);
    }

}
