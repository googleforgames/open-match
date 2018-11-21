<?php

function make_matches($profile_dict, $player_pools) {
    ###########################################################################
    # This is the exciting part, and where most of your custom code would go! #
    ###########################################################################

    foreach ($profile_dict['properties']['rosters'] as &$roster) {
        foreach ($roster['players'] as &$player) {
            if (array_key_exists('pool', $player)) {
                $player['id'] = array_rand($player_pools[$player['pool']]);
                printf("Selected player %s from pool %s (strategy: RANDOM)\n", 
                    $player['id'], 
                    $player['pool']
                );
            } else {
                var_export($player);
            }
        }
        unset($player);
    }
    unset($roster);

    return $profile_dict;
}

?>