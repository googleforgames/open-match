using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Logic.InternalContracts;
using Microsoft.Extensions.Logging;

namespace Logic
{
    /// <summary>
    /// A Match de-collider takes non-colliding Matches in descending score order
    /// </summary>
    public class ScoreEvaluator : IEvaluator
    {
        ILogger<ScoreEvaluator> m_Log { get; }

        public ScoreEvaluator(ILogger<ScoreEvaluator> log)
        {
            m_Log = log;
        }

        public Task<List<Match>> Evaluate(List<Match> Matches)
        {
            m_Log.LogDebug("{MatchCount} Matches to be evaluated", Matches.Count);

            // Sort the Matches by score
            Matches = Matches.OrderByDescending(p => p.Properties["score"]).ToList();

            List<Match> goodMatches = new List<Match>();
            HashSet<Guid> ticketsPresent = new HashSet<Guid>();
            foreach (var nextMatch in Matches)
            {
                // Optimize by converting the prop tickets to a hashset
                var propTickets = nextMatch.Tickets.Select(t => t.Id).ToHashSet();

                // Check if any of the tickets in the Match are already spoken for
                if (ticketsPresent.Overlaps(propTickets))
                    continue;

                // If not, the Match is a good match and mark the tickets as spoken for
                goodMatches.Add(nextMatch);
                foreach (var ticketId in propTickets)
                {
                    ticketsPresent.Add(ticketId);
                }
            }

            m_Log.LogDebug("{MatchesApproved} Matches approved in evaluation. {TicketsApproved} tickets approved", goodMatches.Count, ticketsPresent.Count);

            return Task.FromResult(goodMatches);
        }
    }
}
