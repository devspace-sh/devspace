import React from 'react';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import styles from './ConfigVariablesPortlet.module.scss';
import IconText from 'components/basic/IconText/IconText';
import SettingsIcon from 'images/settings-icon.svg';
import Input from 'components/basic/Input/Input';

interface Props {
  vars: { [key: string]: string };
}

const ConfigVariablesPortlet = (props: Props) => {
  const renderContent = () => {
    return (
      <div className={styles.vars}>
        {!props.vars && <div className={styles['no-vars']}>No variables set</div>}
        {props.vars && (
          <React.Fragment>
            <div className={styles.heading}>
              <div>Variable</div>
              <div>Value</div>
            </div>
            {Object.entries(props.vars).map(([key, value]) => {
              return (
                <div className={styles.var} key={key}>
                  <div className={styles['key-wrapper']}>
                    <label>
                      <Input className={styles.key} disabled={true} value={key} />
                    </label>
                    <span>=</span>
                  </div>
                  <Input className={styles.value} disabled={true} value={value} />
                </div>
              );
            })}
          </React.Fragment>
        )}
      </div>
    );
  };

  return (
    <PortletSimple className={styles['config-var-portlet']}>
      {{
        top: {
          left: (
            <IconText icon={SettingsIcon}>
              <span className={styles['label-value-title']}>Config Variables</span>
            </IconText>
          ),
        },
        content: renderContent(),
      }}
    </PortletSimple>
  );
};

export default ConfigVariablesPortlet;
