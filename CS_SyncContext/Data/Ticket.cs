using System;
using System.Collections.Generic;
using Newtonsoft.Json.Linq;

namespace Data
{
    /// <summary>
    /// Intended to be used as a data model abstraction for handling groups of players organized into indexable tickets
    /// </summary>
    public class Ticket
    {
        /// <summary>
        /// The identifier of the ticket tracked by clients
        /// </summary>
        public Guid Id { get; set; }
        
        /// <summary>
        /// A contract for allowing a backend to provide assignment information
        /// </summary>
        public string Assignment { get; set; }
        
        /// <summary>
        /// Range indexes
        /// </summary>
        public IDictionary<string, double> Attributes { get; set; }

        /// <summary>
        /// The milliseconds in unix utc representing when this ticket was created
        /// </summary>
        public long Created {get;set;}
        
        /// <summary>
        /// Custom data provided by the ticket creator
        /// </summary>
        public JObject Properties { get; set; }
    }
}
