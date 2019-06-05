using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Data;
using Logic.InternalContracts;
using Microsoft.Extensions.Logging;
using Newtonsoft.Json.Linq;

namespace Logic
{
    public class MatchmakingBackend : IMatchmakingBackend
    {
        ITicketData m_TicketData;

        SynchronizationContext m_SyncContext;
        
        ILogger<MatchmakingBackend> m_Logger;

        FunctionClientResolver m_FunctionClientResolver;
        
        public MatchmakingBackend(ITicketData ticketData, ILogger<MatchmakingBackend> logger, FunctionClientResolver resolver, SynchronizationContext syncContext)
        {
            m_TicketData = ticketData;
            m_Logger = logger;
            m_FunctionClientResolver = resolver;
            m_SyncContext = syncContext;
        }

        public async Task<List<Match>> GetMatchesAsync(List<MatchSpec> matchSpecs, CancellationToken cancellationToken)
        {
            // Generate a cancellation time for all the functions. TODO: Make the global timeout configurable
            CancellationToken token = AddTimeCancellationToken(cancellationToken, 60000);
            Guid contextRegistrationId = await m_SyncContext.AcquireContext();

            // Execute functions in parallel
            Stopwatch watch = Stopwatch.StartNew();
            List<Task<IEnumerable<Match>>> tasks = new List<Task<IEnumerable<Match>>>();
            foreach (var matchSpec in matchSpecs)
            {
                IFunctionClient client = m_FunctionClientResolver.GetFunctionClientByTarget(matchSpec.Target);
                m_Logger.LogInformation("Running target {Target} as {Kind}", matchSpec.Target.Name, matchSpec.Target.Kind);
                tasks.Add(Task.Run(() => client.RunAsync(matchSpec, token)));
            }

            // Wait for all the Matches to come back in
            List<Match> Matches = (await Task.WhenAll(tasks)).SelectMany(r => r.AsEnumerable()).ToList();
            m_Logger.LogInformation("Function run time {ElapsedMs}ms. Submitting {MatchCount} Matches for evaluation.", watch.ElapsedMilliseconds, Matches.Count);
            watch.Restart();

            // Send the Matches to the evaluator, which will automatically synchronize the Matches
            List<Match> goodMatches = await m_SyncContext.EvaluateAsync(contextRegistrationId, Matches);
            m_Logger.LogDebug("Evaluator waiting time {ElapsedMs}ms", watch.ElapsedMilliseconds);

            // Tell the data api so it can start de-indexing those players
            List<Match> matches = new List<Match>();
            foreach (var Match in goodMatches)
            {
                matches.Add(new Match()
                {
                    Properties = JObject.FromObject(Match.Properties),
                    Tickets = Match.Tickets,
                    // MatchSpec = null TODO: Maybe re-associate the original matchSpec
                });
            }

            return matches;
        }
        
        static CancellationToken AddTimeCancellationToken(CancellationToken token, int ms)
        {
            CancellationTokenSource cts = new CancellationTokenSource(ms);
            return CancellationTokenSource.CreateLinkedTokenSource(token, cts.Token).Token;
        }
    }
}
