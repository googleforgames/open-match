using System;

namespace Data
{
    /// <summary>
    /// A generic range-based filter for querying data in ITicketData
    /// </summary>
    public class Filter
    {
        public Filter(string key, double min, double max)
        {
            Key = key;
            Min = min;
            Max = max;
        }

        /// <summary>
        /// The attribute to query
        /// </summary>
        public string Key { get; set; }

        /// <summary>
        /// The minimum value [inclusive] of the range 
        /// </summary>
        public double Min { get; set; }

        /// <summary>
        /// The maximum value [exclusive] of the range 
        /// </summary>
        public double Max { get; set; }
    }
}
