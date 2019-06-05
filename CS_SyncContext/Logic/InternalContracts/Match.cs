using System;
using System.Collections.Generic;
using Data;
using Newtonsoft.Json.Linq;

namespace Logic.InternalContracts
{
    public class Match
    {
        public Guid Id { get; set; }
        
        public List<Ticket> Tickets { get; set; }
        
        public JObject Properties { get; set; }
    }
}
