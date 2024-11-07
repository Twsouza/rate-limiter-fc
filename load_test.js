// filename: rate_limiter_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    // Ramp-up from 1 to 20 virtual users (VUs) in 10 seconds
    { duration: '10s', target: 20 },
    // Stay at 20 VUs for 20 seconds
    { duration: '20s', target: 20 },
    // Ramp-down to 0 VUs in 10 seconds
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.1'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE_URL = 'http://web:8080/';

// Tokens to be used in the test
const tokens = [
  { token: '', name: 'No Token (IP-based)' },
  { token: 'abc123', name: 'Token abc123 (Limit 100 req/s)' },
  { token: 'def456', name: 'Token def456 (Limit 70 req/s)' },
  { token: 'ghi789', name: 'Token ghi789 (Default Limit)' },
];

//  Returns a random number between min (inclusive) and max (exclusive)
function getRandomArbitrary(min, max) {
  return Math.random() * (max - min) + min;
}

// Function to perform requests with different tokens
function makeRequest(tokenInfo) {
  let params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  if (tokenInfo.token) {
    params.headers['API_KEY'] = tokenInfo.token;
  }

  let res = http.get(BASE_URL, params);

  // If the response status is 429, set res.error to null
  if (res.status === 429) {
    res.error = null;
    console.warn(`Rate limit exceeded for ${tokenInfo.name}`);
    sleep(1);
  }

  // Check the response status
  let success = check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });

  if (!success) {
    console.error(`Request failed for ${tokenInfo.name}: ${res.status}`);
  }
}

export default function () {
  // Randomly select a token configuration for each virtual user
  let tokenInfo = tokens[getRandomArbitrary(0, tokens.length) | 0];

  makeRequest(tokenInfo);
  // Short sleep to simulate think time
  sleep(0.1);
}
