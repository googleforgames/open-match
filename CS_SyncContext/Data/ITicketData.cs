using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace Data
{
    /// <summary>
    /// An interface for creating, deleting, querying, and updating ticket information
    /// </summary>
    public interface ITicketData
    {
        /// <summary>
        /// Marks the provided ticket ids as ignorable in queries for a period of time
        /// </summary>
        /// <param name="ticketIds">The ticket ids to be ignored by queries</param>
        /// <param name="durationMs"></param>
        Task AwaitingAssignmentAsync(IEnumerable<Guid> ticketIds, long durationMs);

        /// <summary>
        /// Populates the assignment field of the target tickets
        /// </summary>
        /// <param name="ticketIds">A list of ticket ids to be assigned</param>
        /// <param name="assignment">A string containing the assignment of the tickets</param>
        /// <returns></returns>
        Task AssignTicketsAsync(IEnumerable<Guid> ticketIds, string assignment);

        /// <summary>
        /// Executes a collection of queries on the ticket data and returns the union of the results
        /// </summary>
        /// <param name="query">The query to execute</param>
        /// <returns>A collection of tickets union of the results</returns>
        Task<IEnumerable<Ticket>> QueryTicketsAsync(Query query);

        /// <summary>
        /// Creates a ticket and adds it to the ticket datastore
        /// </summary>
        /// <param name="ticket">The ticket to create</param>
        /// <remarks>This may create indexes for the data in the ticket to make it queryable</remarks>
        Task CreateTicketAsync(Ticket ticket);

        /// <summary>
        /// Returns a ticket by id
        /// </summary>
        /// <param name="id">The id of the ticket</param>
        /// <returns>The requested ticket if it exists</returns>
        Task<Ticket> GetTicketAsync(Guid id);

        /// <summary>
        /// Deletes a ticket by id
        /// </summary>
        /// <param name="id">The id of the ticket</param>
        /// <remarks>This may delete indexes for the data in the ticket</remarks>
        Task DeleteTicketAsync(Guid id);
    }
}