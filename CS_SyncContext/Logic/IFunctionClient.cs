using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using Logic.InternalContracts;

namespace Logic
{
    public interface IFunctionClient
    {
        Task<IEnumerable<Match>> RunAsync(MatchSpec config, CancellationToken cancellationToken);
    }
}
