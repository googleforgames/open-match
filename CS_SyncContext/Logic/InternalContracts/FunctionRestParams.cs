using System;
using System.Collections.Generic;
using Newtonsoft.Json.Linq;

namespace Logic.InternalContracts
{
    public class FunctionRestParams
    {
        public JObject Config { get; set; }

        public List<Pool> Pools { get; set; }
    }
}
