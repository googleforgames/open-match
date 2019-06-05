using System;
using System.Collections.Generic;

namespace Data
{
    /// <summary>
    /// Captures a searching behavior for ITicketData 
    /// </summary>
    public class Query
    {
        public Query()
        {

        }

        public Query(List<Filter> filters)
        {
            Filters = filters;
        }
        /// <summary>
        /// A list of hard filters to be applied to the provided searchable attributes
        /// </summary>
        public List<Filter> Filters { get; set; }
    }
}
