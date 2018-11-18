#! /usr/bin/env python3
import random

def makeMatches(profile_dict, player_pools):
    ###########################################################################
    # This is the exciting part, and where most of your custom code would go! #
    ###########################################################################

    # The python3 MMF harness passed this function filtered players and their
    # filtered attributes in the player_pools dictionary.  If we wanted to evaluate
    # other player attributes, we could connect to redis directly and query the
    # players by their ID to get the entire 'properties' player JSON passed in
    # to the frontend API when they entered matchmaking.

    # This basic example just pulls players at random from the specified pools in the 
    # profile.  This just serves to show how the dictionaries are accessed and you 
    # should write your own rigourous logic here.
    for roster in profile_dict['properties']['rosters']:
        for player in roster['players']:
            if 'pool' in player:
                player['id']  = random.choice(list(player_pools[player['pool']]))
                print("Selected player %s from pool %s (strategy: RANDOM)" % (player['id'], player['pool']))
            else:
                print(player)
    return profile_dict
