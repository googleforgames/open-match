using System;
using System.Collections.Generic;
using Data;

namespace Logic.InternalContracts
{
    /// <summary>
    /// A generalized matchmaking "hard" filtering description. Consists of sets of filters
    /// </summary>
    public class Pool
    {
        /// <summary>
        /// A friendly name identifier for the pool
        /// </summary>
        public string Name { get; set; }
        
        /// <summary>
        /// The collection of generic filters for performing query logic
        /// </summary>
        public List<Filter> Filters { get; set; }
    }
}
