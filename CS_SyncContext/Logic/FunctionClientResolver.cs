using System;
using System.Net.Http;
using Data;
using Logic.InternalContracts;
using Microsoft.Extensions.Logging;

namespace Logic
{
    public class FunctionClientResolver
    {
        readonly HttpClient m_HttpClient;

        ILoggerFactory m_LoggerFactory;

        public FunctionClientResolver(IServiceProvider serviceProvider, HttpClient httpClient, ITicketData ticketData, ILoggerFactory loggerFactory)
        {
            m_HttpClient = httpClient;
            m_LoggerFactory = loggerFactory;
        }

        public IFunctionClient GetFunctionClientByTarget(TargetFunction target)
        {
            switch (target.Kind)
            {
                case FunctionKind.Rest:
                    return new FunctionRestClient(m_HttpClient, target, m_LoggerFactory.CreateLogger<FunctionRestClient>());
                default:
                    throw new ArgumentOutOfRangeException();
            }
        }
    }
}
