using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Logic.InternalContracts;

namespace Logic
{
    public interface IEvaluator
    {
        Task<List<Match>> Evaluate(List<Match> Matches);
    }
}
