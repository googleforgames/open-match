#!/usr/bin/env php
<?php
# Step 1 - Package this in a linux container image.

require dirname(__FILE__).'/vendor/autoload.php';

require 'mmf.php';

function dump_pb_message($msg) {
    print($msg->serializeToJsonString() . "\n");
}

$debug = strtolower(trim(getenv('DEBUG'))) == 'true';

# Step 2 - Talk to Redis.  This example uses the MM Logic API in OM to read/write to/from redis.
# Establish grpc channel and make the API client stub
$api_conn_info = sprintf('%s:%s', getenv('OM_MMLOGICAPI_SERVICE_HOST'), getenv('OM_MMLOGICAPI_SERVICE_PORT'));

$mmlogic_api = new Api\MmLogicClient($api_conn_info, [
    'credentials' => Grpc\ChannelCredentials::createInsecure(),
]);

# Step 3 - Read the profile written to the Backend API.
# Get profile from redis
$match_object = new Messages\MatchObject([
    'id' => getenv('MMF_PROFILE_ID')
]);
list($profile_pb, $status) = $mmlogic_api->GetProfile($match_object)->wait();
dump_pb_message($profile_pb);

$profile_dict = json_decode($profile_pb->getProperties(), true);

# Step 4 - Select the player data from Redis that we want for our matchmaking logic.
# Embedded in this profile are JSON representations of the filters for each player pool.
# JsonFilterSet() is able to read those directly.  No need to marhal that
# JSON into the protobuf message format
$player_pools = [];
foreach ($profile_pb->getPools() as $empty_pool) {
    $empty_pool_name = $empty_pool->getName();

    # Dict to hold value-sorted field dictionary for easy retreival of players by value
    $player_pools[$empty_pool_name] = [];
    printf("Retrieving pool '%s'\n", $empty_pool_name);

    if (!$empty_pool->getStats()) {
        $empty_pool->setStats(new Messages\Stats());
    }

    if ($debug) {
        $start = microtime(true);
    }

    # Pool filter results are streamed in chunks as they can be too large to send
    # in one grpc message.  Loop to get them all.
    $call = $mmlogic_api->GetPlayerPool($empty_pool);
    foreach ($call->responses() as $partial_results) {
        if ($partial_results->getStats()) {
            $empty_pool->getStats()->setCount($partial_results->getStats()->getCount());
            $empty_pool->getStats()->setElapsed($partial_results->getStats()->getElapsed());
        }
        print ".\n";
        
        $roster = $partial_results->getRoster();
        if ($roster) {
            foreach ($roster->getPlayers() as $player) {
                if (!array_key_exists($player->getId(), $player_pools[$empty_pool_name])) {
                    $player_pools[$empty_pool_name][$player->getId()] = [];
                }
                foreach ($player->getAttributes() as $attr) {
                    $player_pools[$empty_pool_name][$player->getId()][$attr->getName()] = $attr->getValue();
                }
            }
        }
    }
    if ($debug) {
        $end = microtime(true);
        printf("\n'%s': count %06d | elapsed %0.3f\n", $empty_pool_name, count($player_pools[$empty_pool_name]), $end - $start);
    }
}

#################################################################
# Step 5 - Run custom matchmaking logic to try to find a match
# This is in the file mmf.py
$results = make_matches($profile_dict, $player_pools);
#################################################################

if (debug) {
    print("======= match_properties\n");
    var_export($results);
}

# Generate a MatchObject message to write to state storage with the results in it.
$mo = new Messages\MatchObject([
    'id' => getenv('MMF_PROPOSAL_ID'), 
    'properties' => json_encode($results)
]);
$mo->setPools($profile_pb->getPools());

# Access the rosters in dict form within the properties json.
# It is stored at the key specified in the config file.
$rosters_dict = $results;
foreach (explode('.', getenv('JSONKEYS_ROSTERS')) as $partial_key) {
    $rosters_dict = $rosters_dict[$partial_key] ?? [];
}

# Unmarshal the rosters into the MatchObject 
foreach ($rosters_dict as $roster) {
    $r = new Messages\Roster();
    $r->mergeFromJsonString(json_encode($roster));
    $mo->getRosters() []= $r; 
}

if ($debug) {
    print("======== MMF results:\n");
    dump_pb_message($mo);
}

# Step 6 - Write the outcome of the matchmaking logic back to state storage.
# Step 7 - Remove the selected players from consideration by other MMFs.
# CreateProposal does both of these for you, and some other items as well.
list($result, $status) = $mmlogic_api->CreateProposal($mo)->wait();
printf("======== MMF write to state storage:  %s\n", $result->getSuccess() ? 'true' : 'false');
dump_pb_message($result);

# [OPTIONAL] Step 8 - Export stats about this run.
# TODO 

?>