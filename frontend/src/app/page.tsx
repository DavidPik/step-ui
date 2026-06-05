export default function DashboardPage() {
  return (
    <div className="page page-dashboard">
      <section className="cards-row">
        <div className="card card-status card-status-ok">
          <div className="card-title">CA Status</div>
          <div className="card-value">Online</div>
        </div>

        <div className="card card-status">
          <div className="card-title">Active Provisioners</div>
          <div className="card-value">3</div>
        </div>

        <div className="card card-status card-status-info">
          <div className="card-title">Certificates Issued</div>
          <div className="card-value">256</div>
        </div>

        <div className="card card-status card-status-danger">
          <div className="card-title">Certificates Expired</div>
          <div className="card-value">12</div>
        </div>
      </section>

      <section className="grid-two-columns">
        <div className="card card-table">
          <div className="card-header">
            <h2 className="card-header-title">Recent Certificates</h2>
          </div>
          <div className="card-body">
            <table className="table">
              <thead>
                <tr>
                  <th>Common Name</th>
                  <th>Serial</th>
                  <th>Status</th>
                  <th>Expires</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>www.example.com</td>
                  <td>1234567890ABCDEF</td>
                  <td className="badge badge-ok">Active</td>
                  <td>Aug 20, 2024</td>
                </tr>
                <tr>
                  <td>api.service.local</td>
                  <td>A1B2C3D4E5F6</td>
                  <td className="badge badge-ok">Active</td>
                  <td>Jul 15, 2024</td>
                </tr>
                <tr>
                  <td>test.domain.net</td>
                  <td>98765432109876</td>
                  <td className="badge badge-danger">Revoked</td>
                  <td>Jun 01, 2024</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div className="card card-table">
          <div className="card-header">
            <h2 className="card-header-title">Recent Audit Events</h2>
          </div>
          <div className="card-body">
            <table className="table">
              <thead>
                <tr>
                  <th>Event</th>
                  <th>User</th>
                  <th>Timestamp</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>Certificate Issued</td>
                  <td>admin</td>
                  <td>Today, 10:15 AM</td>
                </tr>
                <tr>
                  <td>Certificate Revoked</td>
                  <td>john.doe</td>
                  <td>Yesterday, 3:42 PM</td>
                </tr>
                <tr>
                  <td>Provisioner Created</td>
                  <td>alice</td>
                  <td>May 10, 2024, 09:27 AM</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>
  );
}
