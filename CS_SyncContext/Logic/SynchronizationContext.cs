using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Data;
using Logic.InternalContracts;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace Logic
{
    /// <summary>
    /// A threadsafe way to run an evaluator. Intended to be used as a singleton, or shared context behind a service
    /// </summary>
    public class SynchronizationContext
    {
        int m_MinRunMs;

        int m_MaxRunMs;

        ITicketData m_TicketData;

        IEvaluator m_Evaluator;

        ILogger<SynchronizationContext> m_Logger;

        ConcurrentDictionary<Guid, List<Match>> m_ContextMatches = new ConcurrentDictionary<Guid, List<Match>>();

        ConcurrentDictionary<Guid, List<Guid>> m_ContextResults = new ConcurrentDictionary<Guid, List<Guid>>();

        ConcurrentDictionary<Guid, bool> m_ExistingContexts = new ConcurrentDictionary<Guid, bool>();

        ManualResetEvent m_NewContextsAvailable = new ManualResetEvent(false);

        ManualResetEvent m_ResultsAvailable = new ManualResetEvent(false);

        bool m_AcceptingMatches = false;

        Timer m_Timer;

        Stopwatch m_Watch = new Stopwatch();

        Task m_EvalTask = null;

        SyncState m_State = SyncState.NotRunning;
        
        object startLock = new object();

        enum SyncState
        {
            NotRunning,
            AcceptingContexts,
            AcceptingMatches,
            Evaluating
        }

        public SynchronizationContext(ILogger<SynchronizationContext> logger, ITicketData ticketData, IEvaluator evaluator, IOptions<SynchronizationOptions> options)
        {
            m_Logger = logger;
            m_MinRunMs = options.Value.MinWindowSizeMs;
            m_MaxRunMs = options.Value.MaxWindowSizeMs;
            m_Evaluator = evaluator;
            m_TicketData = ticketData;

            // TODO: Make the loop event schedulable instead of loop driven
            m_Timer = new Timer(
                UpdateState,
                new AutoResetEvent(true),
                options.Value.StateMachineUpdateMs,
                options.Value.StateMachineUpdateMs
            );
        }

        /// <summary>
        /// Thread safe way to acquire a contextId and register for the evaluator
        /// </summary>
        /// <returns>A contextId</returns>
        public async Task<Guid> AcquireContext()
        {
            Stopwatch watch = Stopwatch.StartNew();
            Guid contextId = await WaitRegisterContextAsync();
            m_Logger.LogDebug("{ElapsedMs}ms to acquire context", watch.ElapsedMilliseconds);
            return contextId;
        }

        /// <summary>
        /// Thread safe way to let the evaluator de-collide the passed in Matches with other contexts
        /// </summary>
        /// <param name="contextId">The id of this context</param>
        /// <param name="Matches">Matches to de-collide</param>
        /// <returns>A list of de-collided Matches</returns>
        /// <exception cref="Exception"></exception>
        public async Task<List<Match>> EvaluateAsync(Guid contextId, List<Match> Matches)
        {
            Stopwatch watch = Stopwatch.StartNew();

            // Try to register the Matches with the machine
            if (TryRegisterMatches(contextId, Matches))
            {
                m_Logger.LogDebug("{ElapsedMs}ms to register Matches", watch.ElapsedMilliseconds);
                watch.Restart();

                // Wait for the machine to run and return my results
                List<Guid> good = await WaitResultsAsync(contextId);
                m_Logger.LogDebug("{ElapsedMs}ms evaluation results available", watch.ElapsedMilliseconds);
                List<Match> goodMatches = new List<Match>();
                foreach (var prop in Matches)
                {
                    if (good.Contains(prop.Id))
                    {
                        goodMatches.Add(prop);
                    }
                }

                return goodMatches;
            }

            throw new Exception("Match registration failed");
        }

        /// <summary>
        /// Attempt to wait for context acquisition to become available and register for one. If the evaluator
        /// is not running, it will set the state machine into a runnable state
        /// </summary>
        /// <returns></returns>
        private Task<Guid> WaitRegisterContextAsync()
        {
            // If the machine isn't started, try to start it
            if (m_State == SyncState.NotRunning)
            {
                lock(startLock)
                {
                    // Make sure this call got the lock in time, otherwise bail
                    if (m_State == SyncState.NotRunning)
                    {
                        // The machine isn't running so clear the current results, any registrations, and any Matches
                        m_ContextResults.Clear();
                        m_ExistingContexts.Clear();
                        m_ContextMatches.Clear();

                        // Allow new contexts to register, allow new Matches, and disallow results reading
                        m_State = SyncState.AcceptingContexts;
                        m_NewContextsAvailable.Set();
                        m_ResultsAvailable.Reset();
                        m_AcceptingMatches = true;
                        m_Watch = Stopwatch.StartNew();
                    }
                }
            }

            m_NewContextsAvailable.WaitOne(5000); // TODO: Make this wait timeout automated
            Guid newId = Guid.NewGuid();
            m_ExistingContexts.TryAdd(newId, false);
            return Task.FromResult(newId);
        }

        private Task<List<Guid>> WaitResultsAsync(Guid contextId)
        {
            m_ResultsAvailable.WaitOne(5000); // TODO: Make this wait timeout automated
            return Task.FromResult(m_ContextResults[contextId]);
        }

        private bool TryRegisterMatches(Guid contextId, List<Match> Matches)
        {
            if (!m_ExistingContexts.ContainsKey(contextId)) return false;
            if (!m_AcceptingMatches) return false;

            m_ExistingContexts[contextId] = true;
            m_ContextResults.TryAdd(contextId, new List<Guid>());

            return m_ContextMatches.TryAdd(contextId, Matches);
        }

        private void UpdateState(object state)
        {
            switch (m_State)
            {
                case SyncState.NotRunning:
                    break;
                case SyncState.AcceptingContexts:
                    if (m_Watch.ElapsedMilliseconds > m_MinRunMs)
                    {
                        m_Logger.LogDebug("Min window passed at {ElapsedMs}ms", m_Watch.ElapsedMilliseconds);
                        m_State = SyncState.AcceptingMatches;
                        m_NewContextsAvailable.Reset();
                        UpdateState(state); // Just go ahead and check the accepting state
                    }

                    break;
                case SyncState.AcceptingMatches:
                    bool maxWindowExceeded = m_Watch.ElapsedMilliseconds > m_MaxRunMs;
                    bool allIn = m_ExistingContexts.Values.All(b => b);
                    if (m_Watch.ElapsedMilliseconds > m_MaxRunMs || m_ExistingContexts.Values.All(b => b))
                    {
                        if (maxWindowExceeded) m_Logger.LogDebug("Max window exceeded. Moving to eval at {ElapsedMs}ms", m_Watch.ElapsedMilliseconds);
                        if (allIn) m_Logger.LogDebug("All contexts reported in. Moving to eval at {ElapsedMs}ms", m_Watch.ElapsedMilliseconds);
                        m_State = SyncState.Evaluating;
                        m_AcceptingMatches = false;
                        m_EvalTask = Task.Run(async () =>
                        {
                            // Run the evaluator in parallel to this state system
                            await RunEvaluation(m_Evaluator);

                            // Once done, clear the other threads to read from the results. Set the machine back to doing nothing
                            m_State = SyncState.NotRunning;
                            m_ResultsAvailable.Set();
                            m_Logger.LogDebug("Evaluation completed at {ElapsedMs}ms", m_Watch.ElapsedMilliseconds);
                            m_Watch.Reset();
                        });
                    }

                    break;
                case SyncState.Evaluating:
                    break;
                default:
                    throw new ArgumentOutOfRangeException();
            }
        }

        private async Task RunEvaluation(IEvaluator evaluator)
        {
            // Create a reverse map of context to Match (to rebuild the results at the end)
            Dictionary<Guid, Guid> MatchIdToContextId = new Dictionary<Guid, Guid>();
            List<Match> allMatches = new List<Match>();
            foreach (var contextMatch in m_ContextMatches)
            {
                foreach (var Match in contextMatch.Value)
                {
                    allMatches.Add(Match);
                    MatchIdToContextId.Add(Match.Id, contextMatch.Key);
                }
            }

            // Run the evaluator
            List<Match> matches = await evaluator.Evaluate(allMatches);

            // Flag the selected tickets as un-queryable for a set period of time. // TODO: Make configurable
            List<Guid> ticketsTaken = matches.SelectMany(m => m.Tickets.Select(t => t.Id)).ToList();
            await m_TicketData.AwaitingAssignmentAsync(ticketsTaken, 60000);

            // Put the match results into the proper results context. TODO: Failure handled good enough by unqueryable timeout?
            foreach (var match in matches)
            {
                Guid contextId = MatchIdToContextId[match.Id];
                m_ContextResults[contextId].Add(match.Id);
            }
        }
    }
}