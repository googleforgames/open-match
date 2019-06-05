using System;

namespace Logic
{
    public class SynchronizationOptions
    {
        public const string SectionName = "SynchronizationOptions";
        
        public int MinWindowSizeMs { get; set; }
        
        public int MaxWindowSizeMs { get; set; }
        
        public int StateMachineUpdateMs { get; set; }
    }
}
