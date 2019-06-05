using System;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Logic.InternalContracts;
using Microsoft.Extensions.Logging;
using Newtonsoft.Json;

namespace Logic
{
    public class FunctionRestClient : IFunctionClient
    {
        HttpClient m_Client;

        string m_Address;

        ILogger<FunctionRestClient> m_Log;

        public FunctionRestClient(HttpClient client, TargetFunction targetFunction, ILogger<FunctionRestClient> log)
        {
            m_Log = log;
            m_Client = client;
            IPHostEntry dns = Dns.GetHostEntry(targetFunction.Name);
            m_Address = dns.AddressList[0].ToString();
        }

        public async Task<IEnumerable<Match>> RunAsync(MatchSpec spec, CancellationToken cancellationToken)
        {
            FunctionRestParams context = new FunctionRestParams()
            {
                Pools = new List<Pool>(),
                Config = spec.Config
            };

            foreach (var specPool in spec.Pools)
            {
                context.Pools.Add(new Pool() { Name = specPool.Key, Filters = specPool.Value });
            }

            string json = JsonConvert.SerializeObject(context);
            string url = "http://" + m_Address + ":8080" + "/api/function";
            m_Log.LogDebug("Calling {Url} with body {Body}", url, json);

            HttpRequestMessage request = new HttpRequestMessage(HttpMethod.Post, url);
            request.Content = new StringContent(json);
            request.Content.Headers.Clear();
            request.Content.Headers.Add("Content-Type", "application/json");

            HttpResponseMessage message = await m_Client.SendAsync(request, cancellationToken);
            if (!message.IsSuccessStatusCode)
            {
                m_Log.LogWarning("{StatusCode} received from function {Url}. {Reason}", message.StatusCode, m_Address, message.ReasonPhrase);
            }

            string body = await message.Content.ReadAsStringAsync();

            return JsonConvert.DeserializeObject<List<Match>>(body);
        }
    }
}
