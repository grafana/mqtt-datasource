import CopyWebpackPlugin from 'copy-webpack-plugin';
import type { Configuration } from 'webpack';
import grafanaConfig, { type Env } from './.config/webpack/webpack.config.ts';

export default async function config(env: Env): Promise<Configuration> {
  const baseConfig = await grafanaConfig(env);
  return {
    ...baseConfig,
    plugins: [
      ...(baseConfig.plugins ?? []),
      new CopyWebpackPlugin({
        patterns: [
          { from: '../pkg/schema/dsconfig.json', to: 'schema/dsconfig.json' },
          { from: '../pkg/schema/schema.gen.json', to: 'schema/v0alpha1.json' },
          { from: '../pkg/schema/settings.gen.json', to: 'schema/v0alpha1/settings.json' },
          { from: '../pkg/schema/settings.examples.gen.json', to: 'schema/v0alpha1/settings.examples.json' },
        ],
      }),
    ],
  };
}
