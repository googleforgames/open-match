using System;
using System.Collections.Generic;
using Data;
using Newtonsoft.Json.Linq;

namespace Logic.InternalContracts
{
    public class MatchSpec
    {
        public TargetFunction Target { get; set; }
        
        public JObject Config { get; set; }
        
        public IDictionary<string, List<Filter>> Pools { get; set; }
    }
}
