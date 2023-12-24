import {
  CoreApp,
  // DataQueryRequest,
  // DataQueryResponse,
  DataSourceInstanceSettings,
  // LiveChannelScope,
} from '@grafana/data';
import { DataSourceWithBackend/*, getGrafanaLiveSrv*/ } from '@grafana/runtime';

// import { defaults } from 'lodash';
// import { Observable, merge } from 'rxjs';

import { RabbitMQQuery, RabbitMQDataSourceOptions, DEFAULT_QUERY } from './types';

export class DataSource extends DataSourceWithBackend<RabbitMQQuery, RabbitMQDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RabbitMQDataSourceOptions>) {
    super(instanceSettings);
  }

  // query(request: DataQueryRequest<RabbitMQQuery>): Observable<DataQueryResponse> {
    
  //   const observables = request.targets.map((target, index) => {
  //     const query = defaults(target, DEFAULT_QUERY);

  //     return getGrafanaLiveSrv().getDataStream({
  //       addr: {
  //         scope: LiveChannelScope.DataSource,
  //         namespace: this.uid,
  //         path: `test-connection`, // this will allow each new query to create a new connection
  //         data: {
  //           ...query,
  //         },
  //       },
  //     });
  //   });

  //   return merge(...observables);
  // }

  getDefaultQuery(_: CoreApp): Partial<RabbitMQQuery> {
    return DEFAULT_QUERY;
  }

  filterQuery(query: RabbitMQQuery): boolean {
    if (query.hide) {
      return false;
    }
    return true;
  }
}
