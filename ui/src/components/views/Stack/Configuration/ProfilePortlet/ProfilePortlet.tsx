import React from 'react';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import styles from './ProfilePortlet.module.scss';
import OwnerIcon from 'images/owner-icon.svg';
import IconText from 'components/basic/IconText/IconText';

interface Props {
  profile: string;
}

const ProfilePortlet = (props: Props) => (
  <PortletSimple className={styles['profile-portlet']}>
    {{
      top: {
        left: (
          <IconText icon={OwnerIcon}>
            <div className={styles['label-value-title']}>
              <span className={styles.label}>Profile</span>
              <span className={styles.value}>{!props.profile ? 'none' : props.profile}</span>
            </div>
          </IconText>
        ),
      },
    }}
  </PortletSimple>
);

export default ProfilePortlet;
