using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

namespace Data
{
    public class MemoryData : ITicketData
    {
        private const string awaitingIndex = "awaitingAssignment";

        private const string createdIndex = "created";

        protected ConcurrentDictionary<Guid, Ticket> m_Tickets = new ConcurrentDictionary<Guid, Ticket>();

        protected ConcurrentDictionary<string, SortedDictionary<Guid, double>> m_Indexes = new ConcurrentDictionary<string, SortedDictionary<Guid, double>>();

        public Task AssignTicketsAsync(IEnumerable<Guid> ticketIds, string assignment)
        {
            foreach (var ticketId in ticketIds)
            {
                Ticket ticket = m_Tickets[ticketId];
                RemoveTicketIndexes(ticket);
                ticket.Assignment = assignment;
            }

            return Task.CompletedTask;
        }

        public Task AwaitingAssignmentAsync(IEnumerable<Guid> ticketIds, long durationMs)
        {
            long unixNowMs = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
            AddOrCreateIndex(ticketIds, new KeyValuePair<string, double>(awaitingIndex, unixNowMs + durationMs));
            return Task.CompletedTask;
        }

        public Task<IEnumerable<Ticket>> QueryTicketsAsync(Query query)
        {
            // Validate the query
            if (query.Filters == null || query.Filters.Count < 1) throw new ArgumentException("Must specify at least 1 filter");
            
            // Get a copy of the indexes and tickets
            Dictionary<string, SortedDictionary<Guid, double>> indexes = new Dictionary<string, SortedDictionary<Guid, double>>();
            foreach (var keyValueIndex in m_Indexes)
            {
                lock (keyValueIndex.Value)
                {
                    indexes.Add(keyValueIndex.Key, new SortedDictionary<Guid, double>(keyValueIndex.Value));
                }
            }

            List<List<Guid>> hits = new List<List<Guid>>();
            foreach (var filter in query.Filters)
            {
                if (indexes.ContainsKey(filter.Key))
                {
                    IEnumerable<Guid> hit = indexes[filter.Key]
                        .Where(i => i.Value >= filter.Min && i.Value <= filter.Max)
                        .Select(k => k.Key);
                    hits.Add(hit.ToList());
                }
            }

            List<Guid> pool = hits.FirstOrDefault();
            if (pool == null)
                return Task.FromResult<IEnumerable<Ticket>>(new List<Ticket>());

            for (int i = 1; i < hits.Count; i++)
            {
                pool = pool.Intersect(hits[i]).ToList();
            }

            // Built-in ignore index
            if (indexes.ContainsKey(awaitingIndex))
            {
                long unixNowMs = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
                IEnumerable<Guid> ignore = indexes[awaitingIndex].Where(i => i.Value >= unixNowMs).Select(k => k.Key);
                pool = pool.Except(ignore).ToList(); // TODO: the list is ordered and except doesn't take advantage of that
            }

            List<Ticket> ticketList = new List<Ticket>();
            foreach (var guid in pool)
            {
                ticketList.Add(m_Tickets[guid]);
            }

            return Task.FromResult<IEnumerable<Ticket>>(ticketList);
        }

        public Task CreateTicketAsync(Ticket ticket)
        {
            // Validate the ticket
            if (ticket == null) throw new ArgumentNullException(nameof(ticket));
            if (ticket.Attributes == null) throw new ArgumentNullException(paramName: "ticketAttributes");
            if (ticket.Attributes.Count == 0) throw new ArgumentException("There must be at least 1 attribute to index", paramName: "ticketAttributes");
            
            Ticket newTicket = new Ticket()
            {
                Id = ticket.Id,
                Attributes = new Dictionary<string, double>(ticket.Attributes),
                Properties = ticket.Properties,
                Created = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
                Assignment = string.IsNullOrEmpty(ticket.Assignment) ? string.Empty : ticket.Assignment
            };

            AddTicketIndexes(newTicket);
            m_Tickets.TryAdd(newTicket.Id, newTicket);
            return Task.CompletedTask;
        }

        public Task<Ticket> GetTicketAsync(Guid id)
        {
            if (m_Tickets.TryGetValue(id, out Ticket ticket))
            {
                return Task.FromResult(ticket);
            }

            throw new Exception("Not Found");
        }

        public Task DeleteTicketAsync(Guid id)
        {
            Ticket ticket = m_Tickets[id];
            RemoveTicketIndexes(ticket);
            m_Tickets.TryRemove(id, out ticket);
            return Task.CompletedTask;
        }

        private void AddTicketIndexes(Ticket ticket)
        {
            foreach (var attribute in ticket.Attributes)
            {
                AddOrCreateIndex(ticket.Id, attribute);
            }

            // Built-in indexes
            AddOrCreateIndex(ticket.Id, new KeyValuePair<string, double>(createdIndex, ticket.Created));
        }

        /// <summary>
        /// The bulk version for reducing locks when updating lots of records at once with the same attribute
        /// </summary>
        private void AddOrCreateIndex(IEnumerable<Guid> ids, KeyValuePair<string, double> attribute)
        {
            if (!m_Indexes.ContainsKey(attribute.Key))
            {
                m_Indexes.TryAdd(attribute.Key, new SortedDictionary<Guid, double>());
            }

            lock (m_Indexes[attribute.Key])
            {
                foreach (var id in ids)
                {
                    m_Indexes[attribute.Key].Add(id, attribute.Value);
                }
            }
        }

        private void AddOrCreateIndex(Guid id, KeyValuePair<string, double> attribute)
        {
            if (m_Indexes.ContainsKey(attribute.Key))
            {
                lock (m_Indexes[attribute.Key])
                {
                    m_Indexes[attribute.Key].Add(id, attribute.Value);
                }
            }
            else
            {
                m_Indexes.TryAdd(attribute.Key, new SortedDictionary<Guid, double>() { { id, attribute.Value } });
            }
        }

        private void RemoveTicketIndexes(Ticket ticket)
        {
            foreach (var attribute in ticket.Attributes)
            {
                DeleteIndex(ticket.Id, attribute.Key);
            }
        }

        private void DeleteIndex(Guid id, string attributeKey)
        {
            if (m_Indexes.ContainsKey(attributeKey))
            {
                lock (m_Indexes[attributeKey])
                {
                    m_Indexes[attributeKey].Remove(id);
                }
            }
        }

        public async Task CreateTicketsAsync(IEnumerable<Ticket> tickets)
        {
            foreach (var ticket in tickets)
            {
                await CreateTicketAsync(ticket);
            }
        }
    }
}
