import React from 'react'
import classnames from 'classnames'

export const Table = ({children}) => (
  <table cellSpacing='0' className='tl monospace collapse ba br2 b--black-10 pv2 ph3 dt--fixed'>{children}</table>
)

export const Th = ({children, ...props}) => (
  <th className='pv2 ph3 tl f7 fw5 ttu sans-serif charcoal-muted bg-near-white' {...props}>{children}</th>
)

export const Td = ({children, ...props}) => {
  const classes = ['pv2', 'ph3', 'fw4', 'f7', 'charcoal', props.truncate ? 'truncate' : '']
  return <td className={classnames(classes)} {...props}>{children}</td>
}
