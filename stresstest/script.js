import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  // 阶段性施压：从 0 到 5000 用户，持续 30秒
  stages: [
    { duration: '30s', target: 500 }, // 先预热到 500
    { duration: '1m', target: 2000 }, // 爬升到 2000
    { duration: '10s', target: 0 },   // 结束
  ],
};

export default function () {
  // 假设这是读取课程列表的接口（走 Redis）
  const res = http.get('http://localhost/api/v1/courses');

  check(res, {
    'status is 200': (r) => r.status === 200,
    // 确保响应时间小于 500ms
    'duration < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(0.1); // 模拟用户思考时间
}