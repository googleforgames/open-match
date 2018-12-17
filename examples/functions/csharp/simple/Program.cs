using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using Newtonsoft.Json;
using StackExchange.Redis;

namespace mmfdotnet
{
    /// <summary>
    /// An example of a simple match function
    /// </summary>
    /// <remarks>
    /// Compatible with example profiles found here: https://github.com/GoogleCloudPlatform/open-match/tree/master/examples/backendclient/profiles
    /// </remarks>
    class Program
    {
        static void Main(string[] args)
        {
            string host = Environment.GetEnvironmentVariable("REDIS_SERVICE_HOST");
            string port = Environment.GetEnvironmentVariable("REDIS_SERVICE_PORT");

            // Single connection to the open match redis cluster
            Console.WriteLine($"Connecting to redis...{host}:{port}");
            StringBuilder builder = new StringBuilder();
            StringWriter writer = new StringWriter(builder);
            ConnectionMultiplexer redis;
            try
            {
                redis = ConnectionMultiplexer.Connect($"{host}:{port}", writer);
            }
            catch (Exception e)
            {
                writer.WriteLine(e);
                throw;
            }
            finally
            {
                writer.Flush();
                Console.WriteLine(writer.GetStringBuilder());
            }

            IDatabase db = redis.GetDatabase();

            try
            {
                FindMatch(db);
            }
            finally
            {
                // Decrement the number of running MMFs since this one is finished
                Console.WriteLine("DECR concurrentMMFs");
                db.StringDecrement("concurrentMMFs");
            }
        }

        private static void FindMatch(IDatabase db)
        {
            // PROFILE is passed via the k8s downward API through an env set to jobName.
            string jobName = Environment.GetEnvironmentVariable("PROFILE");
            Console.WriteLine("PROFILE from job name " + jobName);

            string[] tokens = jobName.Split('.');
            string timestamp = tokens[0];
            string moId = tokens[1];
            string profileKey = tokens[2];

            string resultsKey = $"proposal.{jobName}";
            string rosterKey = $"roster.{jobName}";
            string errorKey = $"{moId}.{profileKey}";

            Console.WriteLine($"Looking for a profile in key " + profileKey);
            string profileJson = db.StringGet(profileKey);

            Profile profile = JsonConvert.DeserializeObject<Profile>(profileJson);

            if (profile.Properties.PlayerPool.Count < 1)
            {
                Console.WriteLine("Insufficient filters");
                db.StringSet(errorKey, "{ \"error\": \"insufficient_filters\"}");
                return;
            }

            // Filter the player pool into sets matching the given filters
            List<List<string>> filteredIds = new List<List<string>>();
            foreach (KeyValuePair<string, string> filter in profile.Properties.PlayerPool)
            {
                string[] range = filter.Value.Split('-');
                int min = int.Parse(range[0]);
                int max = int.Parse(range[1]);
                Console.WriteLine($"Filtering {filter.Key} for {min} to {max}");
                List<string> idsFound = new List<string>();

                // TODO: Only poll a reasonable number (not the whole table)
                RedisValue[] set = db.SortedSetRangeByRank(filter.Key, min, max);
                Console.WriteLine($"Found {set.Count()} matching");
                filteredIds.Add(Array.ConvertAll(set, item => item.ToString()).ToList());
            }

            // Find the union of the player sets (TODO: optimize)
            List<string> overlap = new List<string>();
            foreach (List<string> set in filteredIds)
            {
                overlap = overlap.Union(set).ToList();
            }

            Console.WriteLine($"Overlapping players in pool: {overlap.Count}");

            int rosterSize = profile.Properties.Roster.Values.Sum();
            if (overlap.Count < rosterSize)
            {
                Console.WriteLine("Insufficient players");
                db.StringSet(errorKey, "{ \"error\": \"insufficient_players\"}");
                return;
            }

            // Split the players into teams based on the profile roster information
            Result result = new Result()
            {
                Teams = new Dictionary<string, List<string>>()
            };

            List<string> roster = new List<string>();
            foreach (KeyValuePair<string, int> team in profile.Properties.Roster)
            {
                Console.WriteLine($"Attempting to fill team {team.Key} with {team.Value} players");

                // Only take as many players as are available, or the maximum available
                List<string> group = overlap.Take(team.Value).ToList();
                result.Teams.Add(team.Key, group);

                Console.WriteLine($"Team {team.Key} roster: " + string.Join(" ", group));

                roster.AddRange(group);
            }

            // Write the match object that will be sent back to the DGS
            // In this example, the output is not a modified profile, but rather, just the team rosters
            db.StringSet(resultsKey, JsonConvert.SerializeObject(result));

            // Write the flattened roster that will be sent to the evaluator
            db.StringSet(rosterKey, string.Join(" ", roster));

            // Finally, write the results key to the proposal queue to trigger the evaluation of these results
            string proposalQueueKey = "proposalq";
            db.SetAdd(proposalQueueKey, jobName);
        }
    }
}
