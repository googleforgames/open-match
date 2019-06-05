using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using Logic.InternalContracts;

namespace Logic
{
    public interface IMatchmakingBackend
    {
        Task<List<Match>> GetMatchesAsync(List<MatchSpec> matchSpecs, CancellationToken cancellationToken);
    }
}
