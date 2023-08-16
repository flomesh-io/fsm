/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

((
  config = JSON.decode(pipy.load('config.json')),
  routeRules = {},
  hostRules = null,
) => (
  Object.keys(config?.RouteRules || {}).forEach(
    ports => (
      hostRules = {},
      Object.keys(config.RouteRules[ports] || {}).forEach(
        hosts => (
          hosts.split(',').forEach(
            host => (hostRules[host.trim()] = config.RouteRules[ports][hosts])
          )
        )
      ),
      config.RouteRules[ports] = hostRules
    )
  ),
  Object.keys(config?.RouteRules || {}).forEach(
    ports => (
      ports.split(',').forEach(
        port => (routeRules[port.trim()] = config.RouteRules[ports])
      )
    )
  ),
  config.RouteRules = routeRules,
  {
    config,
    isDebugEnabled: Boolean(config?.Configs?.EnableDebug),
  }
))()
