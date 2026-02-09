import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import grafanaConfig, { Env } from './.config/webpack/webpack.config';

// React 19 renames __SECRET_INTERNALS; dependencies relying on them can break plugin loading, requiring webpack config extensions (Grafana ≥12.3.0 only).
// Extending webpack config will make your plugin incompatible with versions of Grafana earlier than 12.3.0.
const config = async (env: Env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  return merge(baseConfig, {
    externals: ['react/jsx-runtime', 'react/jsx-dev-runtime'],
  });
};

export default config;
