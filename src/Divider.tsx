import { css } from '@emotion/css';
import React from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

export const Divider = () => {
  const styles = useStyles2(getStyles);
  return <hr className={styles.horizontalDivider} />;
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    horizontalDivider: css({
      borderTop: `1px solid ${theme.colors.border.weak}`,
      margin: theme.spacing(2, 0),
      width: '100%',
    }),
  };
};
