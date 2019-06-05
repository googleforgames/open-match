using System;

namespace Logic.InternalContracts
{
    public class TargetFunction
    {
        public string Name { get; set; }

        public string Version { get; set; }

        public FunctionKind Kind { get; set; }
    }
    
    public enum FunctionKind
    {
        None,
        Rest,
        Grpc,
        Memory
    }
}
