

using System.Collections.Generic;

namespace mmfdotnet
{
    /// <summary>
    /// A deserialization target for the simple match function profile example
    /// </summary>
    public class Profile
    {
        public Properties Properties { get; set; }
    }

    public class Properties
    {
        public Dictionary<string, string> PlayerPool { get; set; }

        public Dictionary<string, int> Roster { get; set; }
    }

    /// <summary>
    /// The output of the match function is a collection of team names and contained players
    /// </summary>
    public class Result
    {
        public Dictionary<string, List<string>> Teams { get; set; }
    }
}